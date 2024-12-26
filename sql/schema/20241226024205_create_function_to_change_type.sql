-- +goose Up
-- +goose StatementBegin
ALTER TABLE transactions ADD COLUMN index_new bigint;

CREATE TABLE migration_progress (
    id SERIAL PRIMARY KEY,
    last_update TIMESTAMPTZ DEFAULT now(),
    total_processed BIGINT,
    last_ctid tid,
    total_rows BIGINT
);

CREATE OR REPLACE FUNCTION migrate_index_to_bigint_batch(last_ctid_param tid)
RETURNS TABLE(new_last_ctid tid, rows_updated int) AS $$
DECLARE
    batch_size INT := 500000;
    affected_rows INT;
BEGIN
    WITH batch AS (
        SELECT ctid, block_hash, txid
        FROM transactions
        WHERE index_new IS NULL
        AND ctid > last_ctid_param
        ORDER BY ctid
        LIMIT batch_size
        FOR UPDATE SKIP LOCKED
    )
    UPDATE transactions t
    SET index_new = t.index
    FROM batch
    WHERE t.ctid = batch.ctid;

    GET DIAGNOSTICS affected_rows = ROW_COUNT;

    SELECT MAX(ctid) INTO new_last_ctid
    FROM transactions
    WHERE index_new IS NOT NULL;

    rows_updated := affected_rows;
    RETURN NEXT;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS migrate_index_to_bigint_batch(tid);
DROP TABLE IF EXISTS migration_progress;
ALTER TABLE transactions DROP COLUMN IF EXISTS index_new;
-- +goose StatementEnd
