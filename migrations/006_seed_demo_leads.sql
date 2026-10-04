-- 006: demo leads (Hamburg / Dresden) incl. one duplicate
--
-- DEMO / TEST DATA ONLY - do not run in production. People and phone numbers
-- are made up (emails use example.com); the addresses are real public
-- addresses in the service areas. 10 rows: 9 distinct leads and one duplicate
-- (same email + same address as lead ...0005, linked via duplicate_of).
--
-- Fixed ids make the script safe to run more than once. Normalized columns
-- follow backend/normalization.go (lowercase, ß -> ss, E.164-like phone).

INSERT INTO leads (
    id, first_name, last_name, email, email_normalized, phone, phone_normalized,
    street, street_normalized, house_number, house_number_normalized,
    postal_code, city, city_normalized, parcel_note,
    utm_source, utm_medium, utm_campaign, status, duplicate_of, created_at
) VALUES
    -- Hamburg
    ('00000000-0000-4000-8000-000000000001', 'Lena', 'Hansen', 'lena.hansen@example.com', 'lena.hansen@example.com',
     '+49 40 4281234', '+49404281234', 'Rathausmarkt', 'rathausmarkt', '1', '1',
     '20095', 'Hamburg', 'hamburg', 'Eckgrundstück',
     'meta', 'paid_social', 'standort-check-hamburg', 'new', NULL, NOW() - INTERVAL '9 days'),

    ('00000000-0000-4000-8000-000000000002', 'Jonas', 'Petersen', 'jonas.petersen@example.com', 'jonas.petersen@example.com',
     '+49 170 5550102', '+491705550102', 'Jungfernstieg', 'jungfernstieg', '1', '1',
     '20354', 'Hamburg', 'hamburg', NULL,
     'google', 'cpc', 'standort-check-hamburg', 'contacted', NULL, NOW() - INTERVAL '8 days'),

    ('00000000-0000-4000-8000-000000000003', 'Sabine', 'Krüger', 'sabine.krueger@example.com', 'sabine.krueger@example.com',
     '+49 171 5550103', '+491715550103', 'Mönckebergstraße', 'mönckebergstrasse', '7', '7',
     '20095', 'Hamburg', 'hamburg', 'Grundstück hinter dem Gebäude',
     'meta', 'paid_social', 'standort-check-hamburg', 'qualified', NULL, NOW() - INTERVAL '7 days'),

    ('00000000-0000-4000-8000-000000000004', 'Mehmet', 'Yilmaz', 'mehmet.yilmaz@example.com', 'mehmet.yilmaz@example.com',
     '+49 40 4271234', '+49404271234', 'Osterstraße', 'osterstrasse', '88', '88',
     '20259', 'Hamburg', 'hamburg', NULL,
     'google', 'cpc', 'standort-check-hamburg', 'not_qualified', NULL, NOW() - INTERVAL '6 days'),

    -- Dresden
    ('00000000-0000-4000-8000-000000000005', 'Anja', 'Richter', 'anja.richter@example.com', 'anja.richter@example.com',
     '+49 351 4850105', '+493514850105', 'Sophienstraße', 'sophienstrasse', '1', '1',
     '01067', 'Dresden', 'dresden', 'Flurstück 123/4, Gemarkung Altstadt',
     'meta', 'paid_social', 'standort-check-dresden', 'new', NULL, NOW() - INTERVAL '5 days'),

    ('00000000-0000-4000-8000-000000000006', 'Tobias', 'Lange', 'tobias.lange@example.com', 'tobias.lange@example.com',
     '+49 172 5550106', '+491725550106', 'Theaterplatz', 'theaterplatz', '2', '2',
     '01067', 'Dresden', 'dresden', NULL,
     'google', 'cpc', 'standort-check-dresden', 'contacted', NULL, NOW() - INTERVAL '4 days'),

    ('00000000-0000-4000-8000-000000000007', 'Katrin', 'Böhme', 'katrin.boehme@example.com', 'katrin.boehme@example.com',
     '+49 351 4850107', '+493514850107', 'Prager Straße', 'prager strasse', '10', '10',
     '01069', 'Dresden', 'dresden', NULL,
     'meta', 'paid_social', 'standort-check-dresden', 'qualified', NULL, NOW() - INTERVAL '3 days'),

    ('00000000-0000-4000-8000-000000000008', 'Frank', 'Zimmermann', 'frank.zimmermann@example.com', 'frank.zimmermann@example.com',
     '+49 173 5550108', '+491735550108', 'Taschenberg', 'taschenberg', '2', '2',
     '01067', 'Dresden', 'dresden', NULL,
     'google', 'cpc', 'standort-check-dresden', 'new', NULL, NOW() - INTERVAL '2 days'),

    ('00000000-0000-4000-8000-000000000009', 'Ines', 'Wagner', 'ines.wagner@example.com', 'ines.wagner@example.com',
     '+49 351 4850109', '+493514850109', 'Königsbrücker Straße', 'königsbrücker strasse', '61', '61',
     '01099', 'Dresden', 'dresden', 'Unbebautes Grundstück',
     'meta', 'paid_social', 'standort-check-dresden', 'new', NULL, NOW() - INTERVAL '1 day'),

    -- Duplicate of ...0005: same email, same address (submitted again later,
    -- this time via the Google campaign).
    ('00000000-0000-4000-8000-000000000010', 'Anja', 'Richter', 'anja.richter@example.com', 'anja.richter@example.com',
     '+49 351 4850105', '+493514850105', 'Sophienstraße', 'sophienstrasse', '1', '1',
     '01067', 'Dresden', 'dresden', 'Flurstück 123/4, Gemarkung Altstadt',
     'google', 'cpc', 'standort-check-dresden', 'new', '00000000-0000-4000-8000-000000000005', NOW() - INTERVAL '3 hours')
ON CONFLICT (id) DO NOTHING;
