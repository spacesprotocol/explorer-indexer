-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
CREATE INDEX CONCURRENTLY index_tx_outputs_spender
ON tx_outputs (spender_block_hash, spender_txid, spender_index)

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX index_tx_outputs_spender;
-- +goose StatementEnd
