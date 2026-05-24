import { ShieldCheck } from 'lucide-react'

interface Props {
  isNew: boolean
}

export default function MemberKontaktTab({ isNew }: Props) {
  if (isNew) return null

  return (
    <div className="bg-brand-surface-card rounded-xl shadow border-t-4 border-brand-yellow p-6">
      <div className="flex items-start gap-3">
        <ShieldCheck className="w-5 h-5 text-brand-text-muted shrink-0 mt-0.5" />
        <div>
          <h2 className="font-semibold text-brand-text mb-1">Kontaktdaten verschlüsselt</h2>
          <p className="text-sm text-brand-text-muted">
            Adresse, IBAN und Geburtsdatum sind jetzt verschlüsselt gespeichert und nur im Tab
            <strong className="text-brand-text"> „Sensible Daten"</strong> einsehbar (erfordert Tresor-Passphrase).
          </p>
        </div>
      </div>
    </div>
  )
}
