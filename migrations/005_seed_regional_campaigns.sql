-- 005: seed the four regional campaigns (Hamburg / Dresden x Meta / Google)
--
-- Shows up in the admin "Kampagnen" tab. Safe to run more than once: the
-- unique link constraint from 003 makes repeated inserts a no-op.

INSERT INTO campaigns (name, utm_source, utm_medium, utm_campaign) VALUES
    ('Hamburg – Meta Ads',   'meta',   'paid_social', 'standort-check-hamburg'),
    ('Hamburg – Google Ads', 'google', 'cpc',         'standort-check-hamburg'),
    ('Dresden – Meta Ads',   'meta',   'paid_social', 'standort-check-dresden'),
    ('Dresden – Google Ads', 'google', 'cpc',         'standort-check-dresden')
ON CONFLICT ON CONSTRAINT campaigns_link_unique DO NOTHING;
