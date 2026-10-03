import { lazy, Suspense } from 'react'
import { Navigate, Route, Routes } from 'react-router'
import { PageShell } from './components/PageShell'
import { StandortCheckPage } from './pages/StandortCheckPage'
import { SuccessPage } from './pages/SuccessPage'

// The admin area (incl. the chart library) is loaded only when someone opens
// /admin. Visitors of the public form – mostly on mobile, coming from ads –
// never download it.
const AdminLayout = lazy(() => import('./admin/AdminLayout').then((m) => ({ default: m.AdminLayout })))
const LoginPage = lazy(() => import('./admin/pages/LoginPage').then((m) => ({ default: m.LoginPage })))
const LeadsPage = lazy(() => import('./admin/pages/LeadsPage').then((m) => ({ default: m.LeadsPage })))
const LeadDetailPage = lazy(() => import('./admin/pages/LeadDetailPage').then((m) => ({ default: m.LeadDetailPage })))
const CampaignsPage = lazy(() => import('./admin/pages/CampaignsPage').then((m) => ({ default: m.CampaignsPage })))
const DashboardPage = lazy(() => import('./admin/pages/DashboardPage').then((m) => ({ default: m.DashboardPage })))

export default function App() {
  return (
    <Suspense fallback={<p className="p-8 text-sm text-neutral-500">Lade …</p>}>
      <Routes>
        <Route path="/" element={<Navigate to={{ pathname: '/standort-check', search: window.location.search }} replace />} />
        <Route path="/standort-check" element={<StandortCheckPage />} />
        <Route path="/standort-check/danke" element={<SuccessPage />} />
        <Route path="/datenschutz" element={<Placeholder title="Datenschutzerklärung" text="Platzhalter – hier steht die Datenschutzerklärung des Unternehmens." />} />
        <Route path="/admin/login" element={<LoginPage />} />
        <Route path="/admin" element={<AdminLayout />}>
          <Route index element={<LeadsPage />} />
          <Route path="leads/:id" element={<LeadDetailPage />} />
          <Route path="dashboard" element={<DashboardPage />} />
          <Route path="kampagnen" element={<CampaignsPage />} />
        </Route>
        <Route path="*" element={<Placeholder title="Seite nicht gefunden" text="Diese Seite gibt es nicht." />} />
      </Routes>
    </Suspense>
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
