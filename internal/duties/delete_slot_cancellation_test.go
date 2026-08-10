package duties_test

import (
	"database/sql"
	"net/http"
	"strings"
	"testing"

	appconfig "github.com/teamstuttgart/teamwerk/internal/config"
	"github.com/teamstuttgart/teamwerk/internal/duties"
	"github.com/teamstuttgart/teamwerk/internal/hub"
	"github.com/teamstuttgart/teamwerk/internal/notify"
	"github.com/teamstuttgart/teamwerk/internal/testutil"
)

// ── DeleteSlot: Absage-Benachrichtigung mit Inhalt (openspec absage-benachrichtigung, Abschnitt 6) ──

// sentNotification captures one notify.Send-Aufruf für Assertions in den
// Tests unten.
type sentNotification struct {
	userIDs  []int
	category string
	title    string
	body     string
	url      string
}

// captureSentNotifications überschreibt den notify.Send-Seam und liefert
// einen gepufferten Kanal mit allen Aufrufen (im Gegensatz zu
// captureNotifyCategory in notify_category_test.go, das nur die Kategorie
// mitschneidet — hier brauchen wir Titel/Body/URL für die Textprüfungen).
func captureSentNotifications(t *testing.T) chan sentNotification {
	t.Helper()
	ch := make(chan sentNotification, 16)
	orig := notify.Send
	notify.Send = func(_ *sql.DB, _ *appconfig.Config, userIDs []int, category, title, body, url string) {
		ch <- sentNotification{userIDs: userIDs, category: category, title: title, body: body, url: url}
	}
	t.Cleanup(func() { notify.Send = orig })
	return ch
}

// TestDeleteSlot_CancellationBodyContainsDutyTypeEventDateAndActor prüft, dass
// die Absage-Meldung Dienstart, Event-Name und Datum (TT.MM.JJJJ) enthält und
// der Link weiterhin auf /dienste zeigt — statt des alten Platzhaltertexts.
func TestDeleteSlot_CancellationBodyContainsDutyTypeEventDateAndActor(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	helperID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, helperID, "assigned")

	adminID := testutil.CreateUser(t, db, "admin")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	sent := captureSentNotifications(t)
	token := testutil.Token(t, adminID, "admin", nil)
	res := testutil.Do(t, srv, http.MethodDelete, "/api/duty-slots/"+itoa(slotID), token, nil)
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	select {
	case n := <-sent:
		if n.title != "Dienst abgesagt" {
			t.Errorf("title = %q, want %q", n.title, "Dienst abgesagt")
		}
		if n.url != "/dienste" {
			t.Errorf("url = %q, want /dienste", n.url)
		}
		if !containsAll(n.body, "Aufbau", "Testdienst", "14.06.2026") {
			t.Errorf("body %q enthält nicht Dienstart+Event+Datum", n.body)
		}
	default:
		t.Fatalf("keine Benachrichtigung erhalten")
	}
}

// TestDeleteSlot_CancellationBodyContainsReason prüft, dass ein mitgeschickter
// Grund im Body landet.
func TestDeleteSlot_CancellationBodyContainsReason(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Kasse", 1.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-07-01")

	helperID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, helperID, "assigned")

	adminID := testutil.CreateUser(t, db, "admin")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	sent := captureSentNotifications(t)
	token := testutil.Token(t, adminID, "admin", nil)
	res := testutil.Do(t, srv, http.MethodDelete, "/api/duty-slots/"+itoa(slotID), token,
		map[string]any{"reason": "Dienst wird nicht mehr gebraucht"})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	select {
	case n := <-sent:
		if !containsAll(n.body, "Dienst wird nicht mehr gebraucht") {
			t.Errorf("body %q enthält nicht den Grund", n.body)
		}
	default:
		t.Fatalf("keine Benachrichtigung erhalten")
	}
}

// TestDeleteSlot_VorstandSilent_SuppressesNotificationButBroadcasts prüft,
// dass ein Vorstand mit silent:true keine Benachrichtigung auslöst, das
// SSE-Live-Update aber trotzdem läuft (design.md §8: nicht unterdrückbar).
func TestDeleteSlot_VorstandSilent_SuppressesNotificationButBroadcasts(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	helperID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, helperID, "assigned")

	eh := hub.NewHub()
	h := duties.NewHandler(db, testutil.TestConfig(), eh)
	srv := testServer(t, h)

	// Per-User-Stream des betroffenen Helfers abonnieren — BroadcastToUsers
	// liefert nur an SubscribeUser, nicht an das globale Subscribe().
	ch := eh.SubscribeUser(helperID)
	defer eh.UnsubscribeUser(helperID, ch)

	sent := captureSentNotifications(t)
	vorstandID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, vorstandID, "standard", []string{"vorstand"})
	res := testutil.Do(t, srv, http.MethodDelete, "/api/duty-slots/"+itoa(slotID), token,
		map[string]any{"silent": true})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	select {
	case n := <-sent:
		t.Fatalf("erwartete keine Benachrichtigung, bekam %+v", n)
	default:
	}

	select {
	case ev := <-ch:
		if ev != "duties" {
			t.Errorf("got broadcast %q, want %q", ev, "duties")
		}
	default:
		t.Errorf("kein Broadcast trotz silent — Live-Updates dürfen nicht unterdrückt werden")
	}
}

// TestDeleteSlot_TrainerSilent_NotificationStillSent prüft die Fail-safe-Regel
// aus design.md §4: ein Trainer hat nicht die Capability
// suppress_event_notification, silent wird also stillschweigend ignoriert
// (kein 403), die Benachrichtigung geht trotzdem raus.
func TestDeleteSlot_TrainerSilent_NotificationStillSent(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	helperID := testutil.CreateUser(t, db, "standard")
	insertDutyAssignment(t, db, slotID, helperID, "assigned")

	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	sent := captureSentNotifications(t)
	trainerID := testutil.CreateUser(t, db, "standard")
	token := testutil.Token(t, trainerID, "standard", []string{"trainer"})
	res := testutil.Do(t, srv, http.MethodDelete, "/api/duty-slots/"+itoa(slotID), token,
		map[string]any{"silent": true})
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	select {
	case n := <-sent:
		if n.category != "duties" {
			t.Errorf("category = %q, want duties", n.category)
		}
	default:
		t.Fatalf("erwartete eine Benachrichtigung (Trainer darf silent nicht durchsetzen), bekam keine")
	}
}

// TestDeleteSlot_EmptyBody_StaysBackwardsCompatible prüft, dass ein Request
// ohne Body wie bisher zu HTTP 204 führt (design.md §5: alte PWA-Installationen
// senden keinen Body).
func TestDeleteSlot_EmptyBody_StaysBackwardsCompatible(t *testing.T) {
	db := testutil.NewDB(t)
	seasonID := testutil.CreateSeason(t, db, "2025/26")
	teamID := testutil.CreateTeam(t, db, "Team A")
	dtID := createDutyType(t, db, "Aufbau", 2.0)
	slotID := createDutySlot(t, db, dtID, seasonID, teamID, 0, "2026-06-14")

	adminID := testutil.CreateUser(t, db, "admin")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	token := testutil.Token(t, adminID, "admin", nil)
	res := testutil.Do(t, srv, http.MethodDelete, "/api/duty-slots/"+itoa(slotID), token, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}
	if got := countRows(t, db, "duty_slots", "id=?", slotID); got != 0 {
		t.Errorf("slot not deleted: got %d rows", got)
	}
}

// TestDeleteSlot_UnknownID_404NoNotification prüft, dass eine unbekannte ID
// weiterhin 404 liefert und keine Benachrichtigung auslöst.
func TestDeleteSlot_UnknownID_404NoNotification(t *testing.T) {
	db := testutil.NewDB(t)
	adminID := testutil.CreateUser(t, db, "admin")
	h := duties.NewHandler(db, testutil.TestConfig(), hub.NewHub())
	srv := testServer(t, h)

	sent := captureSentNotifications(t)
	token := testutil.Token(t, adminID, "admin", nil)
	res := testutil.Do(t, srv, http.MethodDelete, "/api/duty-slots/999999", token, nil)
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.StatusCode)
	}

	select {
	case n := <-sent:
		t.Fatalf("erwartete keine Benachrichtigung bei unbekannter ID, bekam %+v", n)
	default:
	}
}

// containsAll prüft, dass s alle needles als Teilstring enthält.
func containsAll(s string, needles ...string) bool {
	for _, n := range needles {
		if !strings.Contains(s, n) {
			return false
		}
	}
	return true
}
