package members

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/teamstuttgart/teamwerk/internal/mailer"
)

const welcomeEmailSubject = "Herzlich Willkommen im Team Stuttgart Handball!"

const welcomeEmailTemplate = `%s %s,

wir freuen uns sehr, Dich im Team Stuttgart Handball willkommen zu heißen! Am %s haben wir Dich unter der Mitgliedsnummer %s im Verein zur Talentförderung des Handballs in Stuttgart e.V. aufgenommen. Anbei findest Du die aktuelle Vereinssatzung, die Gebührenordnung und unser Leitbild als PDF-Datei. Das SEPA-Formular liegt bereits vor, vielen Dank.

Mit Deinem sportlichen Engagement und Deiner Motivation bist Du eine wertvolle Bereicherung für unser Team. Wir sind überzeugt, dass Du sowohl sportlich als auch menschlich hervorragend zu uns passt und freuen uns auf eine erfolgreiche Zusammenarbeit – sowohl auf als auch neben dem Spielfeld.

Ein herzliches Willkommen auch an Deine Familie! Ein unterstützendes Umfeld spielt eine entscheidende Rolle für die sportliche Entwicklung. Wir würden uns freuen, wenn sich Eltern und Angehörige aktiv einbringen – sei es durch Fahrdienste, Unterstützung bei Veranstaltungen und/oder als Mitglieder unseres Fördervereins. Jede Form des Engagements stärkt unsere Gemeinschaft.

Unser Förderverein unterstützt das Team in vielen Bereichen und ermöglicht einen Großteil unserer Aktivitäten. Weitere Informationen sowie das Beitrittsformular findest Du unter: https://www.team-stuttgart.org/foerderverein.

Bei Fragen stehen Dir das Trainerteam und unsere Ansprechpartner jederzeit gerne zur Verfügung.

Willkommen im Team – wir freuen uns auf eine spannende und erfolgreiche Saison!

Mit sportlichen Grüßen,

Team Stuttgart

--
Verein zur Talentförderung des Handballs in Stuttgart e.V.

Team Stuttgart
mail@team-stuttgart.org
www.team-stuttgart.org`

type WelcomeEmailHandler struct {
	db     *sql.DB
	mailer *mailer.Mailer
}

func NewWelcomeEmailHandler(db *sql.DB, m *mailer.Mailer) *WelcomeEmailHandler {
	return &WelcomeEmailHandler{db: db, mailer: m}
}

// POST /api/admin/members/{id}/welcome-email
func (h *WelcomeEmailHandler) Send(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var firstName, lastName, gender, memberNumber string
	var joinDate, welcomeSentAt sql.NullString
	var userEmail sql.NullString

	err := h.db.QueryRowContext(r.Context(), `
		SELECT m.first_name, m.last_name, COALESCE(m.gender,'u'),
		       COALESCE(m.member_number,''), m.join_date, m.welcome_email_sent_at,
		       u.email
		FROM members m
		LEFT JOIN users u ON u.id = m.user_id
		WHERE m.id = ?`, id,
	).Scan(&firstName, &lastName, &gender, &memberNumber, &joinDate, &welcomeSentAt, &userEmail)
	if err != nil {
		http.Error(w, "member not found", http.StatusNotFound)
		return
	}

	if !userEmail.Valid || userEmail.String == "" {
		http.Error(w, "no user account linked", http.StatusBadRequest)
		return
	}
	if welcomeSentAt.Valid && welcomeSentAt.String != "" {
		http.Error(w, "welcome email already sent", http.StatusConflict)
		return
	}

	salutation := "Liebe/r"
	switch gender {
	case "m":
		salutation = "Lieber"
	case "f":
		salutation = "Liebe"
	}

	dateStr := time.Now().Format("02.01.2006")
	if joinDate.Valid && len(joinDate.String) >= 10 {
		if t, err := time.Parse("2006-01-02", joinDate.String[:10]); err == nil {
			dateStr = t.Format("02.01.2006")
		}
	}

	memberNum := memberNumber
	if memberNum == "" {
		memberNum = "–"
	}

	body := fmt.Sprintf(welcomeEmailTemplate, salutation, firstName, dateStr, memberNum)

	attachments, err := loadWelcomeAttachments()
	if err != nil {
		http.Error(w, "failed to load attachments", http.StatusInternalServerError)
		return
	}

	if err := h.mailer.SendWithAttachments(userEmail.String, welcomeEmailSubject, body, attachments); err != nil {
		http.Error(w, "failed to send email: "+err.Error(), http.StatusInternalServerError)
		return
	}

	sentAt := time.Now().UTC().Format(time.RFC3339)
	h.db.ExecContext(r.Context(),
		`UPDATE members SET welcome_email_sent_at = ? WHERE id = ?`, sentAt, id)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"sent_at":"` + sentAt + `"}`))
}

func loadWelcomeAttachments() ([]mailer.Attachment, error) {
	files := []struct {
		path     string
		filename string
		mime     string
	}{
		{"attachments/satzung.pdf", "Vereinssatzung.pdf", "application/pdf"},
		{"attachments/gebuehrenordnung.pdf", "Gebuehrenordnung.pdf", "application/pdf"},
		{"attachments/leitbild.pdf", "Leitbild.pdf", "application/pdf"},
		{"attachments/logo.svg", "TeamStuttgart-Logo.svg", "image/svg+xml"},
	}

	result := make([]mailer.Attachment, 0, len(files))
	for _, f := range files {
		data, err := mailer.AttachmentFS.ReadFile(f.path)
		if err != nil {
			return nil, fmt.Errorf("attachment %s: %w", f.filename, err)
		}
		result = append(result, mailer.Attachment{
			Filename: f.filename,
			Data:     data,
			MIMEType: f.mime,
		})
	}
	return result, nil
}
