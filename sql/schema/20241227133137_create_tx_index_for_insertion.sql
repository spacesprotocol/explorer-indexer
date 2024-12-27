-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
CREATE INDEX CONCURRENTLY index_tx_outputs_unspent
ON tx_outputs (txid, index)
WHERE spender_block_hash IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX index_tx_outputs_unspent;
DROP INDEX index_tx_outputs_spender;
-- +goose StatementEnd
