-- +goose Up
-- +goose StatementBegin
CREATE TABLE tx_outputs  (
    block_hash bytea NOT NULL references blocks (hash) ON DELETE CASCADE,
    txid bytea NOT NULL references transactions (txid) ON DELETE CASCADE, --txid in which output occured
    index bigint NOT NULL CHECK(index >= 0), --index of output inside tx
    value bigint NOT NULL,
    scriptPubKey bytea,
    PRIMARY KEY (block_hash, txid, index)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP table tx_outputs;
-- +goose StatementEnd
