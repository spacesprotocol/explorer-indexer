-- +goose Up
-- +goose StatementBegin
CREATE TABLE commitments (
    block_hash bytea NOT NULL,
    txid bytea NOT NULL,
    name TEXT not null check (length(name) <= 64),
    state_root bytea check (length(state_root)=32),
    history_hash bytea check (length(history_hash)=32),
    revocation boolean not null default false,
    FOREIGN KEY (block_hash, txid) REFERENCES transactions (block_hash, txid) ON DELETE CASCADE 
);

CREATE INDEX idx_commitments_name ON commitments(name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP INDEX idx_commitments_name;
DROP table commitments;
-- +goose StatementEnd
