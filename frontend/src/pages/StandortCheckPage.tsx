import { useEffect } from 'react'
import { captureAttribution } from '../attribution'
import { LeadForm } from '../components/LeadForm'
import { PageShell } from '../components/PageShell'

export function StandortCheckPage() {
  useEffect(() => {
    captureAttribution()
  }, [])

  return (
    <PageShell>
      <h1 className="mb-6 text-center text-2xl font-bold leading-tight sm:mb-10 sm:text-4xl">
        Kostenloser Standort-Check für Ihr Grundstück
      </h1>
      <LeadForm />
    </PageShell>
  )
}
