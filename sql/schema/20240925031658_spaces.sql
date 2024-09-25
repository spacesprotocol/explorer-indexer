-- +goose Up
-- +goose StatementBegin
CREATE TYPE covenant_action
AS ENUM ('RESERVE', 'BID', 'TRANSFER');


CREATE TABLE vmetaouts (
    block_hash bytea NOT NULL references blocks (hash) on delete cascade,
    txid bytea NOT NULL references transactions (txid) on delete cascade,
    "index" bigint NOT NULL,
    outpoint_txid bytea NOT NULL references transactions (txid),
    outpoint_index bigint NOT NULL,
    name string NOT NULL CHECK (LENGTH(name < 64)),
    burn_increment bigint,
    covenant_action covenant_action NOT NULL,
    claim_height bigint,
    expire_height bigint,
    PRIMARY KEY (block_hash, txid, index)
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE vmetaouts;
-- +goose StatementEnd
