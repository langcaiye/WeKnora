-- Add chunk-level business metadata payload to PostgreSQL retriever embeddings.
DO $$
BEGIN
    IF current_setting('app.skip_embedding', true) = 'true' THEN
        RAISE NOTICE 'Skipping migration embeddings metadata (app.skip_embedding=true)';
        RETURN;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'embeddings') THEN
        ALTER TABLE embeddings ADD COLUMN IF NOT EXISTS metadata JSONB;
        RAISE NOTICE '[Migration 000063] Added metadata column to embeddings table';
    ELSE
        RAISE NOTICE '[Migration 000063] embeddings table does not exist, skipping';
    END IF;
END $$;
