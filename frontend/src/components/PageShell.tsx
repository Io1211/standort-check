import type { ReactNode } from 'react'

export function PageShell({ children }: { children: ReactNode }) {
  return (
    <main className="mx-auto max-w-xl px-4 pt-8 pb-16 sm:pt-12">
      <div className="mx-auto mb-8 flex size-24 items-center justify-center rounded-full bg-white sm:size-28">
        <PinIcon />
      </div>
      {children}
    </main>
  )
}

function PinIcon() {
  return (
    <svg aria-hidden="true" viewBox="0 0 24 24" className="size-12 text-ink" fill="none" stroke="currentColor" strokeWidth="1.75">
      <path strokeLinecap="round" strokeLinejoin="round" d="M12 21s-7-6.2-7-11.5a7 7 0 1 1 14 0C19 14.8 12 21 12 21Z" />
      <circle cx="12" cy="9.5" r="2.5" />
    </svg>
  )
}
