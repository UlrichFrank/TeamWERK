import { Link } from 'react-router-dom'
import { Shield, BarChart3, Lock, Mail, ArrowLeft } from 'lucide-react'

// Datenschutzhinweise für TeamWERK.
// Hinweis: Vor Go-Live durch den Vorstand prüfen lassen — Platzhalter (Verantwortlicher, Kontaktdaten)
// durch finale Werte ersetzen.
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
              TeamWERK ist die interne Vereinsverwaltungsplattform von Team Stuttgart (Handball).
              Diese Seite informiert über die Verarbeitung personenbezogener Daten bei der Nutzung
              der Anwendung.
            </p>
          </header>

          <section className="space-y-3">
            <div className="flex items-center gap-2">
              <Shield className="w-5 h-5 text-brand-yellow" />
              <h2 className="text-lg font-semibold text-brand-text">Verantwortlicher</h2>
            </div>
            <p className="text-sm text-brand-text">
              Verein Team Stuttgart e.V.<br />
              <em className="text-brand-text-muted">[Anschrift wird vor Go-Live durch den Vorstand ergänzt]</em><br />
              Kontakt: <a className="text-brand-text underline" href="mailto:vorstand@team-stuttgart.org">vorstand@team-stuttgart.org</a>
            </p>
          </section>

          <section className="space-y-3">
            <div className="flex items-center gap-2">
              <Lock className="w-5 h-5 text-brand-yellow" />
              <h2 className="text-lg font-semibold text-brand-text">Hosting &amp; gespeicherte Daten</h2>
            </div>
            <p className="text-sm text-brand-text">
              Die Anwendung wird auf einem Server bei <strong>IONOS</strong> (Deutschland) betrieben.
              Es werden ausschließlich Daten verarbeitet, die für die Vereinsverwaltung notwendig sind:
              Mitgliedsstammdaten, Team- und Kader-Zuordnungen, Spiel- und Trainingsplanung, Dienstplanung,
              vereinsinterne Kommunikation sowie für den Beitragseinzug benötigte Bankdaten.
            </p>
            <p className="text-sm text-brand-text">
              Authentifizierungsdaten (Zugangstokens) werden ausschließlich innerhalb der Anwendung
              genutzt und nicht an Dritte weitergegeben.
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
              Sie haben jederzeit das Recht auf Auskunft, Berichtigung, Löschung oder Einschränkung
              der Verarbeitung Ihrer personenbezogenen Daten sowie ein Recht auf Datenübertragbarkeit
              und Widerspruch. Wenden Sie sich dafür an den Vorstand.
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
