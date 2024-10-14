-- +goose Up
-- +goose StatementBegin
CREATE TABLE tx_inputs (
    block_hash bytea NOT NULL references blocks (hash) on delete cascade,
    txid bytea NOT NULL references transactions (txid) on delete cascade,
    "index" bigint NOT NULL,
    hash_prevout bytea CHECK (hash_prevout IS NULL OR LENGTH(hash_prevout) = 32),
    index_prevout bigint NOT NULL, -- txin_witness bytea NOT NULL CHECK (LENGTH(txin_witness) = 32),
    "sequence" bigint NOT NULL,
    coinbase bytea,
    txinwitness bytea[],
    PRIMARY KEY (block_hash, txid, index),
    UNIQUE (txid, "index"),
    CONSTRAINT coinbase_or_not CHECK ((hash_prevout IS NOT NULL AND index_prevout IS NOT NULL) OR coinbase IS NOT NULL)
);

CREATE INDEX tx_inputs_txid_index ON tx_inputs USING btree (txid);

--to select inputs belonging to some given transaction
CREATE INDEX tx_inputs_hash_prevout_index ON tx_inputs USING btree (hash_prevout, index_prevout) WHERE hash_prevout IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX tx_inputs_txid_index;
DROP INDEX tx_inputs_hash_prevout_index;
DROP TABLE tx_inputs;
-- +goose StatementEnd
