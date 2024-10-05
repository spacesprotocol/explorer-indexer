-- +goose Up
-- +goose StatementBegin
CREATE TYPE covenant_action
AS ENUM ('RESERVE', 'BID', 'TRANSFER');


CREATE TABLE vmetaouts (
    block_hash bytea NOT NULL references blocks (hash) on delete cascade,
    txid bytea NOT NULL references transactions (txid) on delete cascade,
    tx_index bigint NOT NULL CHECK(tx_index >= 0),
    outpoint_txid bytea NOT NULL references transactions (txid),
    outpoint_index bigint NOT NULL CHECK(outpoint_index >= 0),
    name TEXT NOT NULL CHECK (LENGTH(name) < 64),
    burn_increment bigint,
    covenant_action covenant_action NOT NULL,
    claim_height bigint,
    expire_height bigint,
    PRIMARY KEY (block_hash, txid, tx_index)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TYPE covenant_action;
DROP TABLE vmetaouts;
-- +goose StatementEnd
