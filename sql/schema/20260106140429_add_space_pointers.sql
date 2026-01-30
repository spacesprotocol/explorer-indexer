-- +goose Up
-- +goose StatementBegin
CREATE SEQUENCE space_pointers_identifier_seq;

CREATE TABLE space_pointers (
    identifier BIGINT NOT NULL DEFAULT nextval('space_pointers_identifier_seq'::regclass),
    block_hash BYTEA NOT NULL,
    txid BYTEA NOT NULL,
    vout INTEGER NOT NULL CHECK(vout >= 0),  
    sptr TEXT NOT NULL CHECK(length(sptr) = 63), 
    value BIGINT NOT NULL check(value >= 0), 
    script_pubkey BYTEA NOT NULL,
    data BYTEA,

    spent_block_hash BYTEA,
    spent_txid BYTEA,
    spent_vin INTEGER,

    PRIMARY KEY (identifier),
    UNIQUE(block_hash, txid, vout),
    FOREIGN KEY (block_hash, txid) REFERENCES transactions(block_hash, txid) ON DELETE CASCADE,
    FOREIGN KEY (spent_block_hash, spent_txid) REFERENCES transactions(block_hash, txid) ON DELETE CASCADE
);

CREATE INDEX idx_space_pointers_sptr ON space_pointers(sptr);
CREATE INDEX idx_space_pointers_script_pubkey ON space_pointers(script_pubkey);
CREATE INDEX idx_space_pointers_block_hash ON space_pointers(block_hash);


CREATE SEQUENCE sptr_delegations_identifier_seq;

CREATE TABLE sptr_delegations (
    identifier BIGINT NOT NULL DEFAULT nextval('sptr_delegations_identifier_seq'::regclass),
    sptr TEXT NOT NULL CHECK(length(sptr) = 63),
    name TEXT NOT NULL CHECK(length(name) <= 64),

    block_hash BYTEA NOT NULL,
    txid BYTEA NOT NULL,
    vout INTEGER NOT NULL, 

    revoked BOOLEAN NOT NULL DEFAULT false,
    revoked_block_hash BYTEA,
    revoked_txid BYTEA,
    revoked_vout integer check (revoked_vout >= 0),

    PRIMARY KEY (identifier),
    UNIQUE(block_hash, txid, vout),
    FOREIGN KEY (block_hash, txid) REFERENCES transactions(block_hash, txid) ON DELETE CASCADE,
    FOREIGN KEY (revoked_block_hash, revoked_txid) REFERENCES transactions(block_hash, txid) ON DELETE CASCADE
);

CREATE INDEX idx_sptr_delegations_name ON sptr_delegations(name);
CREATE INDEX idx_sptr_delegations_sptr ON sptr_delegations(sptr);
CREATE INDEX idx_sptr_delegations_block_hash ON sptr_delegations(block_hash, txid);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sptr_delegations CASCADE;
DROP TABLE IF EXISTS space_pointers CASCADE;
DROP SEQUENCE IF EXISTS sptr_delegations_identifier_seq;
DROP SEQUENCE IF EXISTS space_pointers_identifier_seq;
-- +goose StatementEnd
