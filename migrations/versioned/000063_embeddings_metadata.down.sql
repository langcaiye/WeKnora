DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'embeddings') THEN
        ALTER TABLE embeddings DROP COLUMN IF EXISTS metadata;
    END IF;
END $$;
