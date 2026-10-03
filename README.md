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

## Confirmation email

After a valid form submission is stored, the API sends a German confirmation
to the submitted email address through Brevo. Configure `BREVO_API_KEY` and
`EMAIL_FROM` in `.env` for local development and in the Vercel environment for
deployment. Use a sender address registered in Brevo:
[Brevo setup and API documentation](https://developers.brevo.com/docs/send-a-transactional-email).

The send runs before the response with a three-second timeout so it also works
in the serverless runtime. Mail failures are logged; the saved request still
succeeds and the thank-you page reports that the email could not be sent.
Without both settings, mail is disabled. There is no automatic retry queue.
Invalid submissions, failed saves and honeypot submissions send no email.

## Geo enrichment (Geoapify)

After a lead is saved, the plot address is geocoded with
[Geoapify](https://www.geoapify.com) (structured search, restricted to Germany).
Stored per lead (migration `004`): status, municipality, county, federal state,
coordinates, the postcode and address the geocoder found, result type and
confidence.

- **Service area:** `SERVICE_AREA_STATES` lists the federal states served. A
  lead is "im Einzugsgebiet" if its geocoded state is in that list. It is
  computed on read, so changing the list applies to all existing leads.
- **Data-quality signals:** "PLZ passt nicht" when the geocoder places the
  address in a different postcode than entered; "Lage ungenau" when only the
  street/town was found or the confidence is below 0.7.
- **Never blocks a lead:** geocoding runs after the save with a 3-second
  timeout. Failures are stored as `error` and can be retried per lead or with
  "Fehlende Geodaten abrufen" on the dashboard (15 leads per click, rate
  limited).
- **Privacy:** only the plot address is sent to Geoapify (a German company),
  never name, email or phone. The API key never appears in logs.

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
