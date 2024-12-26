-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 
        FROM transactions 
        WHERE index_new IS NULL
    ) THEN
        RAISE EXCEPTION 'Migration incomplete: some rows have NULL index_new';
    END IF;
END $$;

DROP INDEX transactions_block_hash_index;
ALTER TABLE transactions DROP COLUMN index;
ALTER TABLE transactions RENAME COLUMN index_new TO index;
ALTER TABLE transactions ALTER COLUMN index SET NOT NULL;
DROP FUNCTION migrate_index_to_bigint_batch(tid);
DROP TABLE migration_progress;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS transactions_block_hash_index;
ALTER TABLE transactions RENAME COLUMN index TO index_new;
ALTER TABLE transactions ADD COLUMN index integer;
UPDATE transactions SET index = index_new::integer;
ALTER TABLE transactions ALTER COLUMN index SET NOT NULL;
ALTER TABLE transactions DROP COLUMN index_new;
-- +goose StatementEnd
