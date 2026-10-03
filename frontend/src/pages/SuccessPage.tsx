import { PageShell } from '../components/PageShell'

export function SuccessPage() {
  return (
    <PageShell>
      <div className="rounded-md bg-white px-6 py-10 text-center sm:px-10">
        <div className="mx-auto mb-6 flex size-14 items-center justify-center rounded-full bg-cream">
          <svg aria-hidden="true" viewBox="0 0 24 24" className="size-8 text-ink" fill="none" stroke="currentColor" strokeWidth="2.5">
            <path strokeLinecap="round" strokeLinejoin="round" d="M5 12.5l4.5 4.5L19 7.5" />
          </svg>
        </div>
        <h1 className="mb-4 text-2xl font-bold sm:text-3xl">Vielen Dank für Ihre Anfrage!</h1>
        <p className="leading-relaxed">
          Wir haben die Angaben zu Ihrem Grundstück erhalten. Unser Team prüft den Standort und meldet sich in Kürze
          telefonisch oder per E-Mail bei Ihnen.
        </p>
        <p className="mt-4 text-sm text-ink/70">Eine Bestätigung haben wir Ihnen per E-Mail geschickt.</p>
      </div>
    </PageShell>
  )
}
