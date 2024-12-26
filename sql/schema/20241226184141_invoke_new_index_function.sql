-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
DO $$
DECLARE
    batch_result INT;
    total_processed BIGINT := 0;
    total_rows BIGINT;
    attempts INT := 0;
BEGIN
    SELECT count(*) INTO total_rows FROM transactions;
    
    -- Begin tracking
    INSERT INTO migration_progress (processed_rows, total_rows) 
    VALUES (0, total_rows);
    COMMIT;

    LOOP
        BEGIN
            SELECT migrate_index_to_bigint_batch() INTO batch_result;
            
            IF batch_result = 0 THEN
                attempts := attempts + 1;
                IF attempts >= 3 THEN  -- Try a few times before giving up
                    -- Double check if we're really done
                    IF EXISTS (
                        SELECT 1 
                        FROM transactions 
                        WHERE index_new IS NULL
                    ) THEN
                        RAISE EXCEPTION 'Found unprocessed rows while attempting to exit';
                    END IF;
                    EXIT;
                END IF;
                -- Small sleep before retry
                PERFORM pg_sleep(1);
                CONTINUE;
            END IF;
            
            -- Reset attempts counter on successful update
            attempts := 0;
            
            total_processed := total_processed + batch_result;
            
            -- Log progress
            INSERT INTO migration_progress (processed_rows, total_rows)
            VALUES (total_processed, total_rows);
            
            COMMIT;
        END;
            
        PERFORM pg_sleep(1);
    END LOOP;

    -- Final verification
    IF EXISTS (
        SELECT 1 
        FROM transactions 
        WHERE index_new IS NULL
    ) THEN
        RAISE EXCEPTION 'Migration incomplete: found unprocessed rows after completion';
    END IF;
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

