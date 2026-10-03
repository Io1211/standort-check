import { PageShell } from '../components/PageShell'
import { useLocation } from 'react-router'

export function SuccessPage() {
  const state = useLocation().state as { confirmationSent?: boolean } | null
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
        {state?.confirmationSent === true && (
          <p className="mt-4 text-sm text-ink/70">Eine Bestätigung haben wir Ihnen per E-Mail geschickt. Bitte prüfen Sie auch Ihren Spam-Ordner.</p>
        )}
        {state?.confirmationSent === false && (
          <p className="mt-4 text-sm text-ink/70">Ihre Anfrage ist gespeichert. Die Bestätigung per E-Mail konnte gerade nicht versendet werden. Sie müssen das Formular nicht erneut absenden.</p>
        )}
      </div>
    </PageShell>
  )
}
