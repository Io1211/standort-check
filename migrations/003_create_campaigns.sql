-- 003: campaigns (tracking link management)
--
-- A campaign is the configuration behind one tracking link. Leads are not
-- linked by foreign key: they keep the UTM values they arrived with (a lead
-- must never be lost or changed because a campaign was renamed or deleted).
-- The dashboard matches them by utm_source + utm_campaign (+ utm_content).

CREATE TABLE campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    name TEXT NOT NULL,

    -- Lowercase slugs only, so "Google" and "google" can never become two
    -- different sources in the evaluation.
    utm_source   TEXT NOT NULL CHECK (utm_source   ~ '^[a-z0-9][a-z0-9._-]*$'),
    utm_medium   TEXT NOT NULL CHECK (utm_medium   ~ '^[a-z0-9][a-z0-9._-]*$'),
    utm_campaign TEXT NOT NULL CHECK (utm_campaign ~ '^[a-z0-9][a-z0-9._-]*$'),
    utm_content  TEXT,          -- optional: ad / variant

    notes TEXT,
    archived BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- One link per combination. NULLS NOT DISTINCT (Postgres 15+) treats two
    -- links without utm_content as equal.
    CONSTRAINT campaigns_link_unique
        UNIQUE NULLS NOT DISTINCT (utm_source, utm_medium, utm_campaign, utm_content)
);

-- Same as leads: block access through Supabase's public REST API.
ALTER TABLE campaigns ENABLE ROW LEVEL SECURITY;
