-- 004: geo enrichment (Geoapify)
--
-- Filled after the lead is stored. All columns are nullable: a lead without
-- geo data is still a complete lead (geocoding is a bonus and may fail).
-- geo_status NULL = not attempted yet.

ALTER TABLE leads
    ADD COLUMN geo_status TEXT CHECK (geo_status IN ('ok', 'not_found', 'error')),
    ADD COLUMN geo_lat DOUBLE PRECISION,
    ADD COLUMN geo_lon DOUBLE PRECISION,
    ADD COLUMN geo_municipality TEXT,   -- Gemeinde / Stadt
    ADD COLUMN geo_county TEXT,         -- Landkreis
    ADD COLUMN geo_state TEXT,          -- Bundesland
    ADD COLUMN geo_postcode TEXT,       -- postcode the geocoder found (may differ from the input)
    ADD COLUMN geo_formatted TEXT,      -- full address as the geocoder understood it
    ADD COLUMN geo_result_type TEXT,    -- building / street / postcode / city …
    ADD COLUMN geo_confidence REAL,     -- 0..1
    ADD COLUMN geo_checked_at TIMESTAMPTZ;
