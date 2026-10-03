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

## SonarCloud analysis

The GitHub Actions workflow `.github/workflows/sonarqube.yml` runs Go tests with
coverage and analyzes the Go backend and React/TypeScript frontend for the
SonarCloud project `Io1211_standort-check`.

In GitHub repository **Settings → Secrets and variables → Actions**, configure:

- Repository secret `SONAR_TOKEN`: the token created in SonarCloud.
- Repository variable `SONAR_ORGANIZATION`: the exact `sonar.organization` value
  shown in the project's SonarCloud GitHub Actions setup.

If Automatic Analysis is enabled in SonarCloud, disable it under
**Administration → Analysis Method** before using this CI workflow.

Push the configuration to `main` to run the analysis, or start **SonarCloud**
manually from the GitHub **Actions** tab. Pull requests also trigger analysis.
The token is read from GitHub Actions secrets; a local `.env` is not used by CI.
