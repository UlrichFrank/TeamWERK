import { useVault } from '../contexts/VaultContext'
import VaultPassphraseDialog from './VaultPassphraseDialog'

interface Props {
  children: React.ReactNode
}

export default function VaultGate({ children }: Props) {
  const { isUnlocked } = useVault()

  if (!isUnlocked) {
    return <VaultPassphraseDialog />
  }

  return <>{children}</>
}
