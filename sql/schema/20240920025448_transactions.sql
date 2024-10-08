-- +goose Up
-- +goose StatementBegin
CREATE TABLE transactions (
    txid bytea PRIMARY KEY CHECK (LENGTH(txid) = 32),
    tx_hash bytea CHECK (LENGTH(tx_hash) = 32),
    -- witness_tx bytea CHECK (LENGTH(witness_tx) = 32) NOT NULL, --wtxid, witness data of transaction
    "version" integer NOT NULL,
    "size" bigint NOT NULL,
    vsize bigint NOT NULL,
    weight bigint NOT NULL,
    -- rate bigint NOT NULL,
    locktime integer NOT NULL,
    fee bigint NOT NULL,
    block_hash bytea REFERENCES blocks (hash) ON DELETE CASCADE,
    "index" integer
);

CREATE UNIQUE INDEX transactions_block_hash_index ON transactions (block_hash, index) WHERE block_hash IS NOT NULL;


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX transactions_block_hash_index;
DROP TABLE transactions;
-- +goose StatementEnd
