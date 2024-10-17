-- +goose Up
-- +goose StatementBegin
CREATE TYPE covenant_action
AS ENUM ('RESERVE', 'BID', 'TRANSFER');


CREATE TABLE vmetaouts (
    block_hash bytea NOT NULL,
    txid bytea NOT NULL,
    tx_index bigint NOT NULL,
    outpoint_txid bytea NOT NULL,
    outpoint_index bigint NOT NULL CHECK(outpoint_index >= 0),
    name TEXT NOT NULL CHECK (LENGTH(name) < 64),
    burn_increment bigint,
    covenant_action covenant_action NOT NULL,
    claim_height bigint,
    expire_height bigint,
    PRIMARY KEY (block_hash, txid, tx_index),
    FOREIGN KEY (block_hash, txid) REFERENCES transactions (block_hash, txid) ON DELETE CASCADE
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE vmetaouts;
DROP TYPE covenant_action;
-- +goose StatementEnd
