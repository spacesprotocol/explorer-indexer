-- +goose Up
-- +goose StatementBegin
CREATE TYPE covenant_action
AS ENUM ('RESERVE', 'BID', 'TRANSFER', 'ROLLOUT', 'REVOKE');

CREATE SEQUENCE vmetaouts_identifier_seq;

CREATE TABLE vmetaouts (
    block_hash bytea NOT NULL,
    txid bytea NOT NULL,

    identifier bigint PRIMARY KEY DEFAULT nextval('vmetaouts_identifier_seq'),

    priority bigint,  --it's the priority of rollouts
    
    name TEXT CHECK (LENGTH(name) < 64),
    reason TEXT, -- revoke reason
    value bigint NOT NULL,
    scriptPubKey bytea NOT NULL,

    action covenant_action,
    burn_increment bigint,
    signature bytea,
    total_burned bigint,

    claim_height bigint,
    expire_height bigint,

    script_error text,

    FOREIGN KEY (block_hash, txid) REFERENCES transactions (block_hash, txid) ON DELETE CASCADE 
);
-- Index for querying by name (where name is not null)
CREATE INDEX index_vmetaouts_name ON vmetaouts(name) WHERE name IS NOT NULL;
CREATE INDEX index_vmetaouts_blockhash_txid ON vmetaouts(block_hash, txid);
CREATE INDEX index_vmetaouts_scriptpubkey ON vmetaouts(scriptPubKey);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX index_vmetaouts_scriptpubkey;
DROP INDEX index_vmetaouts_name;
DROP TABLE vmetaouts;
DROP SEQUENCE vmetaouts_identifier_seq;
DROP TYPE covenant_action;
-- +goose StatementEnd
