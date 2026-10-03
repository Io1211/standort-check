-- 002: deleting a lead must not be blocked by its duplicates.
--
-- The API re-links duplicates itself when a lead is deleted (see
-- Repository.DeleteLead). This is the safety net for deletes made directly in
-- the database (e.g. Supabase table editor): references are cleared instead of
-- the delete failing with a foreign key violation.

ALTER TABLE leads DROP CONSTRAINT leads_duplicate_of_fkey;

ALTER TABLE leads
    ADD CONSTRAINT leads_duplicate_of_fkey
    FOREIGN KEY (duplicate_of) REFERENCES leads(id) ON DELETE SET NULL;
