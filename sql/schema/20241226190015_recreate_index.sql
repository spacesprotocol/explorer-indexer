-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE INDEX CONCURRENTLY transactions_block_hash_index ON transactions (block_hash, index);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX CONCURRENTLY IF EXISTS transactions_block_hash_index;
-- +goose StatementEnd

