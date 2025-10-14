-- +goose Up
-- +goose NO TRANSACTION
SET statement_timeout = 0;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_transactions_mempool
ON transactions(block_hash)
WHERE block_hash = '\xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef';

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_transactions_mempool;
