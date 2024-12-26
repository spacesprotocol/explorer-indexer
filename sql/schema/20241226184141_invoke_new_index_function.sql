-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
DO $$
DECLARE
    last_ctid tid := '(0,0)'::tid;
    batch_result RECORD;
    total_processed BIGINT := 0;
    total_rows BIGINT;
BEGIN
    SELECT count(*) INTO total_rows FROM transactions;
    
    -- Begin tracking
    INSERT INTO migration_progress (total_processed, last_ctid, total_rows) 
    VALUES (0, '(0,0)', total_rows);
    COMMIT;

    LOOP
        BEGIN  -- Start new transaction for each batch
            SELECT * INTO batch_result 
            FROM migrate_index_to_bigint_batch(last_ctid);
            
            EXIT WHEN batch_result.rows_updated = 0;
            
            total_processed := total_processed + batch_result.rows_updated;
            last_ctid := batch_result.new_last_ctid;
            
            -- Log progress
            INSERT INTO migration_progress (total_processed, last_ctid, total_rows)
            VALUES (total_processed, last_ctid, total_rows);
            
            RAISE NOTICE 'Processed % of % rows (%.2f%%), last ctid: %', 
                total_processed, 
                total_rows,
                (total_processed::float * 100 / total_rows),
                last_ctid;
                
            COMMIT;
        END;
            
        PERFORM pg_sleep(0.1);
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DO $$
BEGIN
    UPDATE transactions SET index_new = NULL;
    TRUNCATE migration_progress;
    COMMIT;
END $$;
-- +goose StatementEnd
