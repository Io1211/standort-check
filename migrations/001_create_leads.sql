-- 001: leads table
--
-- Raw user input and normalized values are stored side by side: the raw value
-- is what the customer typed (shown to sales), the normalized value is only
-- used for duplicate detection. Normalized columns are NOT NULL DEFAULT '' so
-- duplicate comparison is a plain equality check (no NULL semantics).

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE leads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Contact
    first_name TEXT NOT NULL,
    last_name  TEXT NOT NULL,

    email            TEXT NOT NULL,
    email_normalized TEXT NOT NULL,

    phone            TEXT NOT NULL,
    phone_normalized TEXT NOT NULL,          -- e.g. +4940123456

    -- Plot / property address
    street            TEXT NOT NULL,
    street_normalized TEXT NOT NULL,

    -- Optional: undeveloped plots often have no house number yet.
    house_number            TEXT,
    house_number_normalized TEXT NOT NULL DEFAULT '',

    postal_code TEXT NOT NULL CHECK (postal_code ~ '^[0-9]{5}$'),

    city            TEXT NOT NULL,
    city_normalized TEXT NOT NULL,

    -- Free text, e.g. "Flurstück 123/4, Gemarkung Klotzsche" or "Eckgrundstück".
    parcel_note TEXT,

    -- Campaign attribution (all optional)
    utm_source   TEXT,
    utm_medium   TEXT,
    utm_campaign TEXT,
    utm_content  TEXT,
    utm_term     TEXT,
    gclid        TEXT,                       -- Google Ads click id
    fbclid       TEXT,                       -- Meta click id
    referrer     TEXT,

    -- Sales workflow
    status TEXT NOT NULL DEFAULT 'new'
        CHECK (status IN ('new', 'contacted', 'qualified', 'not_qualified')),

    -- Points to the oldest matching lead. Duplicates are stored, never rejected.
    duplicate_of UUID REFERENCES leads(id),

    -- Also the time of GDPR consent: the form cannot be submitted without it.
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Duplicate lookup: narrow by address first, then compare email / phone.
CREATE INDEX idx_leads_address ON leads (
    postal_code,
    street_normalized,
    house_number_normalized,
    city_normalized
);

-- Dashboard: default sort, filters, "has duplicates" lookup.
CREATE INDEX idx_leads_created_at   ON leads (created_at DESC);
CREATE INDEX idx_leads_status       ON leads (status);
CREATE INDEX idx_leads_utm_source   ON leads (utm_source);
CREATE INDEX idx_leads_utm_campaign ON leads (utm_campaign);
CREATE INDEX idx_leads_duplicate_of ON leads (duplicate_of);

-- Supabase exposes every table in the public schema through its REST API
-- (PostgREST) using the public anon key. Enabling RLS without any policy
-- blocks that path completely. The Go API connects as the table owner and
-- bypasses RLS, so all access goes through our backend.
ALTER TABLE leads ENABLE ROW LEVEL SECURITY;
