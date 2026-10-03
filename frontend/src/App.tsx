import { Navigate, Route, Routes } from 'react-router'
import { PageShell } from './components/PageShell'
import { StandortCheckPage } from './pages/StandortCheckPage'
import { SuccessPage } from './pages/SuccessPage'

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to={{ pathname: '/standort-check', search: window.location.search }} replace />} />
      <Route path="/standort-check" element={<StandortCheckPage />} />
      <Route path="/standort-check/danke" element={<SuccessPage />} />
      <Route path="/datenschutz" element={<Placeholder title="Datenschutzerklärung" text="Platzhalter – hier steht die Datenschutzerklärung des Unternehmens." />} />
      <Route path="/admin" element={<Placeholder title="Admin" text="Folgt im nächsten Schritt." />} />
      <Route path="*" element={<Placeholder title="Seite nicht gefunden" text="Diese Seite gibt es nicht." />} />
    </Routes>
  )
}

function Placeholder({ title, text }: { title: string; text: string }) {
  return (
    <PageShell>
      <h1 className="mb-4 text-center text-3xl font-bold">{title}</h1>
      <p className="text-center">{text}</p>
    </PageShell>
  )
}
