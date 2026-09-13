import { Link } from 'react-router-dom'
import {
  Shield,
  ShieldCheck,
  Users,
  Landmark,
  HeartPulse,
  Video,
  MessageCircle,
  Baby,
  BarChart3,
  Lock,
  Mail,
  ArrowLeft,
} from 'lucide-react'

export default function DatenschutzPage() {
  return (
    <div className="min-h-screen bg-brand-gray">
      <div className="max-w-3xl mx-auto px-4 py-8 sm:py-12">
        <Link
          to="/login"
          className="inline-flex items-center gap-2 text-sm text-brand-text-muted hover:text-brand-text mb-6"
        >
          <ArrowLeft className="w-4 h-4" />
          Zurück zur Anmeldung
        </Link>

        <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6 sm:p-8 space-y-8">
          <header>
            <h1 className="text-2xl sm:text-3xl font-bold text-brand-text">Datenschutzerklärung</h1>
            <p className="text-sm text-brand-text-muted mt-2">
              TeamWERK ist die interne Vereinsverwaltungsplattform des Vereins zur Talentförderung
              des Handballs in Stuttgart e.V. (Team Stuttgart). Diese Seite informiert über die
              Verarbeitung personenbezogener Daten bei der Nutzung der Anwendung unter
              teamwerk.team-stuttgart.org. Für die öffentliche Vereins-Homepage
              www.team-stuttgart.org gilt die dortige eigene Datenschutzerklärung.
            </p>
          </header>

          <section className="space-y-3">
            <div className="flex items-center gap-2">
              <Shield className="w-5 h-5 text-brand-yellow" />
              <h2 className="text-lg font-semibold text-brand-text">Verantwortlicher</h2>
            </div>
            <p className="text-sm text-brand-text">
              Verein zur Talentförderung des Handballs in Stuttgart e.V.<br />
              c/o Marko Baisch<br />
              Alosenweg 75<br />
              70329 Stuttgart<br />
              Telefon: 0711 31 02 560<br />
              E-Mail: <a className="text-brand-text underline" href="mailto:vorstand@team-stuttgart.org">vorstand@team-stuttgart.org</a>
            </p>
          </section>

          <section className="space-y-3">
            <div className="flex items-center gap-2">
              <Users className="w-5 h-5 text-brand-yellow" />
              <h2 className="text-lg font-semibold text-brand-text">Mitgliedsdaten und Vereinsverwaltung</h2>
            </div>
            <p className="text-sm text-brand-text">
              Zur Verwaltung der Vereinsmitgliedschaft und des Spielbetriebs verarbeiten wir
              Stammdaten (Name, Geburtsdatum, Kontaktdaten, Anschrift), Mitgliedsstatus und
              Ein-/Austrittsdatum, Team- und Kader-Zuordnungen, Vereinsfunktionen (z. B. Trainer,
              Vorstand), Spiel- und Trainingsplanung, Dienstplanung (z. B. Kuchen-, Kassen- und
              Schiedsrichterdienste) sowie die dafür nötigen Rückmeldungen und Anwesenheiten.
              Rechtsgrundlage ist die Erfüllung der Mitgliedschaft im Verein (Art. 6 Abs. 1 lit. b
              DSGVO) bzw. berechtigtes Interesse an einem funktionierenden Spiel- und
              Dienstbetrieb (Art. 6 Abs. 1 lit. f DSGVO).
            </p>
            <p className="text-sm text-brand-text">
              Für den Spielplan gleichen wir Ansetzungen des Verbands (Handball4All/BWHV) ab.
              Dabei übermittelte Zugangsdaten eines gemeinsamen Vereins-Accounts werden
              ausschließlich für die Dauer des jeweiligen Abrufs verwendet und weder protokolliert
              noch gespeichert.
            </p>
          </section>

          <section className="space-y-3">
            <div className="flex items-center gap-2">
              <ShieldCheck className="w-5 h-5 text-brand-yellow" />
              <h2 className="text-lg font-semibold text-brand-text">Einwilligungen</h2>
            </div>
            <p className="text-sm text-brand-text">
              Für Mitglieder erfassen wir gesondert die Einwilligung in die Datenverarbeitung zur
              Vereinsverwaltung, in die Weitergabe von Mitgliedsdaten an Dritte (z. B. Verband,
              Versicherung), soweit für den Vereinsbetrieb erforderlich, sowie in die
              Veröffentlichung von Fotos auf öffentlichen Kanälen des Vereins (Homepage,
              Spielberichte). Jede Einwilligung kann im eigenen Profil unter „Datenschutz" jederzeit
              geändert oder widerrufen werden; Änderungen prüft und übernimmt der Vorstand.
            </p>
          </section>

          <section className="space-y-3">
            <div className="flex items-center gap-2">
              <Landmark className="w-5 h-5 text-brand-yellow" />
              <h2 className="text-lg font-semibold text-brand-text">Bankdaten (SEPA-Beitragseinzug)</h2>
            </div>
            <p className="text-sm text-brand-text">
              Bankverbindungen für den Mitgliedsbeitrag (IBAN, Kontoinhaber) werden
              <strong> Ende-zu-Ende clientseitig verschlüsselt</strong>: Ihr Browser verschlüsselt
              die Daten, bevor sie den Server erreichen. Der Server speichert ausschließlich
              Chiffrat und besitzt selbst keinen Schlüssel, um es zu entschlüsseln. Entschlüsseln
              können nur Vorstand und Kassierer, die dafür einen gemeinsamen, ausschließlich im
              Browser gehaltenen Tresor-Schlüssel entsperren. Damit ist ausgeschlossen, dass
              Bankdaten bei einem Datenbank-Zugriff, -Backup oder durch den Hosting-Anbieter im
              Klartext einsehbar sind. Diese Verschlüsselung schützt nicht vor einem kompromittierten
              Endgerät der entschlüsselnden Person. Ein Verlust des Tresor-Kennworts durch alle
              Inhaber führt zum unwiederbringlichen Verlust der Bankdaten; sie müssten erneut
              erfasst werden. Rechtsgrundlage ist die Erfüllung der Beitragspflicht aus der
              Mitgliedschaft (Art. 6 Abs. 1 lit. b DSGVO). Der SEPA-Lastschrifteinzug selbst
              erfolgt einmal jährlich zum 1. Juli der Saison.
            </p>
          </section>

          <section className="space-y-3">
            <div className="flex items-center gap-2">
              <HeartPulse className="w-5 h-5 text-brand-yellow" />
              <h2 className="text-lg font-semibold text-brand-text">Trainingstagebuch</h2>
            </div>
            <p className="text-sm text-brand-text">
              Spieler:innen können freiwillig ein eigenes Trainingstagebuch führen (Datum, Art und
              Dauer der Einheit, subjektive Belastung, optional ein Nachweisfoto). Diese Einträge
              sind ausschließlich für das jeweilige Mitglied selbst, dessen Eltern sowie
              Trainer:innen und sportliche Leitung des zugehörigen Kaders einsehbar — nicht für den
              Vorstand. Nachweisfotos werden 90 Tage nach Saisonende automatisch gelöscht, die
              übrigen Angaben bleiben als Historie erhalten. Rechtsgrundlage ist die Einwilligung
              durch die freiwillige Nutzung (Art. 6 Abs. 1 lit. a, bei Minderjährigen durch die
              Erziehungsberechtigten).
            </p>
          </section>

          <section className="space-y-3">
            <div className="flex items-center gap-2">
              <Video className="w-5 h-5 text-brand-yellow" />
              <h2 className="text-lg font-semibold text-brand-text">Spielvideos und Fotos</h2>
            </div>
            <p className="text-sm text-brand-text">
              Von Heimspielen können Videoaufzeichnungen hochgeladen werden. Der Zugriff auf ein
              Video ist auf Mitglieder, Eltern und Trainer:innen der beteiligten Mannschaften sowie
              Vorstand beschränkt; ein Streaming-Link ist zeitlich begrenzt gültig. Eine
              Veröffentlichung von Fotos außerhalb der Anwendung (z. B. auf der Vereins-Homepage
              oder in Spielberichten) erfolgt ausschließlich mit der gesondert erteilten Einwilligung
              zur Foto-Veröffentlichung (siehe „Einwilligungen").
            </p>
          </section>

          <section className="space-y-3">
            <div className="flex items-center gap-2">
              <MessageCircle className="w-5 h-5 text-brand-yellow" />
              <h2 className="text-lg font-semibold text-brand-text">Kommunikation</h2>
            </div>
            <p className="text-sm text-brand-text">
              Für die vereinsinterne Kommunikation stehen ein Chat zwischen Mitgliedern sowie
              Push- und E-Mail-Benachrichtigungen zu Terminen, Diensten und Nachrichten zur
              Verfügung. Push-Benachrichtigungen setzen eine gerätebezogene Einwilligung im
              Browser voraus und lassen sich jederzeit in den Profileinstellungen oder im Browser
              widerrufen. E-Mails werden über den eigenen Mailserver des Hosting-Anbieters
              versendet, nicht über einen Drittanbieter-Dienst.
            </p>
          </section>

          <section className="space-y-3">
            <div className="flex items-center gap-2">
              <Baby className="w-5 h-5 text-brand-yellow" />
              <h2 className="text-lg font-semibold text-brand-text">Kinder-Accounts und Eltern</h2>
            </div>
            <p className="text-sm text-brand-text">
              Minderjährige Mitglieder erhalten einen eigenen Zugang. Über eine Eltern-Kind-Verknüpfung
              können Erziehungsberechtigte die Daten ihres Kindes einsehen und in dessen Namen
              handeln (z. B. Termin-Rückmeldungen, Bankdaten pflegen), jedoch keine Bankdaten
              zurücklesen, die bereits verschlüsselt gespeichert sind. Die Einwilligungen zur
              Datenverarbeitung, Datenweitergabe und Foto-Veröffentlichung werden bei Minderjährigen
              durch die Erziehungsberechtigten erteilt.
            </p>
          </section>

          <section className="space-y-3">
            <div className="flex items-center gap-2">
              <Lock className="w-5 h-5 text-brand-yellow" />
              <h2 className="text-lg font-semibold text-brand-text">Hosting, Sicherheit und Speicherdauer</h2>
            </div>
            <p className="text-sm text-brand-text">
              Die Anwendung wird auf einem Server bei <strong>IONOS</strong> (Deutschland)
              betrieben, die Verbindung ist ausschließlich über HTTPS erreichbar. Zugangstoken
              werden ausschließlich innerhalb der Anwendung genutzt und nicht an Dritte
              weitergegeben. Tägliche Datensicherungen werden 14 Tage vorgehalten. Mitgliedsdaten
              werden für die Dauer der Mitgliedschaft und darüber hinaus nur verarbeitet, soweit
              gesetzliche Aufbewahrungspflichten (insbesondere handels- und steuerrechtlich) dies
              erfordern; danach werden die Daten gelöscht oder anonymisiert.
            </p>
          </section>

          <section className="space-y-3">
            <div className="flex items-center gap-2">
              <BarChart3 className="w-5 h-5 text-brand-yellow" />
              <h2 className="text-lg font-semibold text-brand-text">Anonyme Nutzungsstatistiken (Matomo)</h2>
            </div>
            <p className="text-sm text-brand-text">
              Zur Verbesserung der Anwendung erfassen wir anonyme Nutzungsdaten über
              <strong> Matomo</strong>, eine Open-Source-Lösung für Web-Analytics. Die Matomo-Instanz
              wird im Auftrag des Vereins bei <strong>mittwald CM Service GmbH &amp; Co. KG</strong>
              (Deutschland) betrieben. Ein Auftragsverarbeitungsvertrag liegt vor.
            </p>
            <p className="text-sm text-brand-text">
              Erfasst werden ausschließlich folgende Informationen:
            </p>
            <ul className="list-disc list-inside text-sm text-brand-text space-y-1 ml-2">
              <li>aufgerufene Seiten innerhalb von TeamWERK (Pfade, ohne Inhalte)</li>
              <li>verwendeter Kanal: installierte App (PWA) oder normaler Browser</li>
              <li>grobes Team-Segment (Kurzbezeichnung des Haupt-Teams, ohne Personenbezug)</li>
              <li>Rollen-Segment (Administrator oder Standard-Nutzer)</li>
              <li>vom Browser automatisch übermittelte technische Daten (User-Agent, Sprache, Auflösung)</li>
            </ul>
            <p className="text-sm text-brand-text">
              Die IP-Adresse wird vor der Speicherung um die letzten zwei Oktette gekürzt
              (Anonymisierung). Es werden <strong>keine Cookies</strong> für das Tracking gesetzt
              und <strong>keine Nutzer-IDs, E-Mail-Adressen oder Klarnamen</strong> übermittelt.
              Die Browser-Einstellung &bdquo;Do Not Track&ldquo; wird respektiert — ist sie aktiv,
              findet keine Erfassung statt.
            </p>
            <p className="text-sm text-brand-text">
              <strong>Kinder-Accounts</strong> werden in dieser Statistik genauso anonym behandelt
              wie Erwachsenen-Accounts. Es werden keine personenbezogenen Daten zu Kindern an
              Matomo übermittelt.
            </p>
            <p className="text-sm text-brand-text-muted">
              Rechtsgrundlage: berechtigtes Interesse (Art. 6 Abs. 1 lit. f DSGVO) an der
              technischen Verbesserung der Vereinsanwendung.
            </p>
          </section>

          <section className="space-y-3">
            <h2 className="text-lg font-semibold text-brand-text">Ihre Rechte</h2>
            <p className="text-sm text-brand-text">
              Sie haben das Recht auf Auskunft über die zu Ihrer Person gespeicherten Daten
              (Art. 15 DSGVO), auf Berichtigung unrichtiger Daten (Art. 16), auf Löschung
              (Art. 17), auf Einschränkung der Verarbeitung (Art. 18), auf Datenübertragbarkeit
              (Art. 20) sowie auf Widerspruch gegen eine auf berechtigtem Interesse beruhende
              Verarbeitung (Art. 21). Wenden Sie sich dafür an den Vorstand. Zusätzlich haben Sie
              das Recht, sich bei einer Datenschutz-Aufsichtsbehörde zu beschweren — für
              Baden-Württemberg ist dies der Landesbeauftragte für den Datenschutz und die
              Informationsfreiheit Baden-Württemberg (LfDI BW).
            </p>
          </section>

          <section className="space-y-3">
            <div className="flex items-center gap-2">
              <Mail className="w-5 h-5 text-brand-yellow" />
              <h2 className="text-lg font-semibold text-brand-text">Kontakt</h2>
            </div>
            <p className="text-sm text-brand-text">
              Bei Fragen zum Datenschutz wenden Sie sich bitte an
              {' '}
              <a className="text-brand-text underline" href="mailto:vorstand@team-stuttgart.org">
                vorstand@team-stuttgart.org
              </a>.
            </p>
          </section>
        </div>
      </div>
    </div>
  )
}
