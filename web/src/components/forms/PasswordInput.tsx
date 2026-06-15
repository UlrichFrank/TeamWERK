import { useRef, useState, ChangeEvent } from 'react'
import { Eye, EyeOff } from 'lucide-react'

interface Props {
  value: string
  onChange: (value: string) => void
  autoComplete: 'current-password' | 'new-password' | 'off'
  required?: boolean
  id?: string
  name?: string
  placeholder?: string
  autoFocus?: boolean
  disabled?: boolean
  minLength?: number
}

const INPUT_CLASS =
  'w-full border border-brand-border rounded-md px-3 py-2 pr-10 text-sm text-brand-text placeholder:text-brand-text-subtle focus:outline-none focus:ring-2 focus:ring-brand-yellow focus:border-brand-yellow'

const USER_ACTION_WINDOW_MS = 100

export default function PasswordInput({
  value,
  onChange,
  autoComplete,
  required,
  id,
  name,
  placeholder,
  autoFocus,
  disabled,
  minLength,
}: Props) {
  const [revealed, setRevealed] = useState(false)
  const [userTyped, setUserTyped] = useState(false)
  const [tainted, setTainted] = useState(false)
  const lastUserActionAt = useRef(0)

  const markUserAction = () => {
    lastUserActionAt.current = performance.now()
  }

  const handleChange = (e: ChangeEvent<HTMLInputElement>) => {
    const next = e.target.value
    if (next === '') {
      setUserTyped(false)
      setTainted(false)
    } else if (performance.now() - lastUserActionAt.current < USER_ACTION_WINDOW_MS) {
      setUserTyped(true)
    } else {
      setTainted(true)
    }
    onChange(next)
  }

  const allowReveal = userTyped && !tainted && value.length > 0

  return (
    <div className="relative">
      <input
        type={revealed ? 'text' : 'password'}
        value={value}
        onChange={handleChange}
        onKeyDown={markUserAction}
        onPaste={markUserAction}
        onCut={markUserAction}
        onBlur={() => setRevealed(false)}
        autoComplete={autoComplete}
        required={required}
        id={id}
        name={name}
        placeholder={placeholder}
        autoFocus={autoFocus}
        disabled={disabled}
        minLength={minLength}
        className={INPUT_CLASS}
      />
      {allowReveal && (
        <button
          type="button"
          onClick={() => setRevealed((r) => !r)}
          aria-label={revealed ? 'Passwort verbergen' : 'Passwort anzeigen'}
          aria-pressed={revealed}
          className="absolute right-0 top-0 h-full px-3 flex items-center text-brand-text-muted hover:text-brand-text"
        >
          {revealed ? <EyeOff className="w-5 h-5" /> : <Eye className="w-5 h-5" />}
        </button>
      )}
    </div>
  )
}
