-- +goose Up
-- +goose StatementBegin
ALTER TABLE transactions ADD COLUMN index_new bigint;

CREATE TABLE migration_progress (
    id SERIAL PRIMARY KEY,
    last_update TIMESTAMPTZ DEFAULT now(),
    processed_rows BIGINT,
    total_rows BIGINT
);

CREATE OR REPLACE FUNCTION migrate_index_to_bigint_batch()
RETURNS int AS $$
DECLARE
    batch_size INT := 500000;
    affected_rows INT;
BEGIN
    WITH batch AS (
        SELECT block_hash, txid
        FROM transactions
        WHERE index_new IS NULL
        LIMIT batch_size
        FOR UPDATE SKIP LOCKED
    )
    UPDATE transactions t
    SET index_new = t.index
    FROM batch b
    WHERE t.block_hash = b.block_hash 
    AND t.txid = b.txid;

    GET DIAGNOSTICS affected_rows = ROW_COUNT;
    RETURN affected_rows;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS migrate_index_to_bigint_batch();
DROP TABLE IF EXISTS migration_progress;
ALTER TABLE transactions DROP COLUMN IF EXISTS index_new;
-- +goose StatementEnd

