-- +goose Up
-- +goose StatementBegin
CREATE TABLE transactions (
    txid bytea NOT NULL CHECK (LENGTH(txid) = 32),
    tx_hash bytea NOT NULL CHECK (LENGTH(tx_hash) = 32),
    "version" integer NOT NULL,
    "size" bigint NOT NULL,
    vsize bigint NOT NULL,
    weight bigint NOT NULL,
    locktime integer NOT NULL,
    fee bigint NOT NULL,
    block_hash bytea REFERENCES blocks (hash) ON DELETE CASCADE,
    "index" integer,

    PRIMARY KEY (block_hash, txid),
    UNIQUE (block_hash, txid)
);

CREATE UNIQUE INDEX transactions_block_hash_index ON transactions (block_hash, index);
CREATE INDEX transactions_txid_index ON transactions (txid);


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX transactions_block_hash_index;
DROP TABLE transactions;
-- +goose StatementEnd
