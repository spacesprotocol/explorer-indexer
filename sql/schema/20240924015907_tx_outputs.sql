-- +goose Up
-- +goose StatementBegin
CREATE TABLE tx_outputs  (
    block_hash bytea NOT NULL,
    txid bytea NOT NULL,
    index bigint NOT NULL CHECK(index >= 0),
    value bigint NOT NULL,
    scriptPubKey bytea NOT NULL,
    spender_txid bytea,
    spender_index bigint,
    PRIMARY KEY (block_hash, txid, "index"),
    FOREIGN KEY (block_hash, txid) REFERENCES transactions (block_hash, txid) ON DELETE CASCADE,
    CONSTRAINT fk_spender FOREIGN KEY (spender_txid, spender_index) REFERENCES tx_inputs (txid, "index") DEFERRABLE INITIALLY DEFERRED
);

-- Create an index on (txid, index) for faster lookups
CREATE INDEX idx_tx_outputs_txid_index ON tx_outputs (txid, index);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_tx_outputs_txid_index;
DROP TABLE tx_outputs;
-- +goose StatementEnd
