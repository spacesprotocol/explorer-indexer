-- +goose Up
-- +goose NO TRANSACTION
CREATE INDEX idx_blocks_hash_not_orphan ON blocks(hash)
WHERE NOT orphan;


-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_blocks_hash_not_orphan;
