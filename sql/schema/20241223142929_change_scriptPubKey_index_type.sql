-- +goose NO TRANSACTION
-- +goose Up
-- +goose StatementBegin
DROP INDEX IF EXISTS index_tx_outputs_scriptpubkey;
-- +goose StatementEnd

CREATE INDEX CONCURRENTLY index_tx_outputs_scriptpubkey ON tx_outputs USING hash (sha256(scriptpubkey));

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS index_tx_outputs_scriptpubkey;
-- +goose StatementEnd
