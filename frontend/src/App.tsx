import { Navigate, Route, Routes } from 'react-router'

// Placeholder routes. The real pages are added step by step.
export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/standort-check" replace />} />
      <Route path="/standort-check" element={<Placeholder title="Standort-Check" />} />
      <Route path="/admin" element={<Placeholder title="Admin" />} />
      <Route path="*" element={<Placeholder title="Seite nicht gefunden" />} />
    </Routes>
  )
}

function Placeholder({ title }: { title: string }) {
  return (
    <main className="mx-auto max-w-xl p-6">
      <h1 className="text-2xl font-semibold">{title}</h1>
    </main>
  )
}
