# Standort-Check

Lead form for a free plot location check plus an internal sales dashboard.
Case study for Planeco Building GmbH.

```
Browser ──▶ Vercel ──┬─▶ React (Vite, static)
                     └─▶ /api/* ──▶ Go function (chi) ──▶ Supabase PostgreSQL
```

> Work in progress – this README is extended step by step.

## Repository structure

```
api/index.go          Vercel serverless entry point (wraps the chi router)
backend/              All business logic (package backend)
cmd/server/           Local development server (same router)
frontend/             React + TypeScript + Vite + Tailwind
migrations/           SQL migrations, run manually in Supabase SQL editor
vercel.json           Build config + rewrites (/api/* → Go, everything else → SPA)
```

## Run locally

Requirements: Go ≥ 1.25, Node ≥ 20.

```bash
cp .env.example .env        # fill in DATABASE_URL at minimum
go run ./cmd/server         # API on http://localhost:8080
```

```bash
cd frontend
npm install
npm run dev                 # http://localhost:5173, proxies /api → :8080
```

Health check: `GET /api/health` returns `{"status":"ok","database":"ok"}`.
