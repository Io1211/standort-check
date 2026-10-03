import type { InputHTMLAttributes } from 'react'

interface FieldProps extends InputHTMLAttributes<HTMLInputElement> {
  id: string
  label: string
  error?: string
}

// Input with an attached error box, styled like the campaign landing pages.
// The label is visually hidden but available to screen readers.
export function Field({ id, label, error, className = '', ...input }: FieldProps) {
  return (
    <div className={`overflow-hidden rounded-md bg-white focus-within:ring-2 focus-within:ring-accent/50 ${className}`}>
      <label htmlFor={id} className="sr-only">
        {label}
      </label>
      <input
        id={id}
        name={id}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? `${id}-error` : undefined}
        className="w-full min-w-0 bg-transparent px-5 py-4 text-base text-ink outline-none placeholder:text-neutral-400"
        {...input}
      />
      <FieldError id={`${id}-error`} message={error} />
    </div>
  )
}

export function FieldError({ id, message }: { id: string; message?: string }) {
  if (!message) return null
  return (
    <p id={id} className="flex items-center gap-3 bg-error-bg px-5 py-3 text-sm text-error">
      <svg aria-hidden="true" viewBox="0 0 24 24" className="size-4 shrink-0" fill="none" stroke="currentColor" strokeWidth="2.5">
        <path d="M6 6l12 12M18 6L6 18" strokeLinecap="round" />
      </svg>
      {message}
    </p>
  )
}
