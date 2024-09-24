-- +goose Up
-- +goose StatementBegin
CREATE TABLE blocks (
    hash bytea PRIMARY KEY CHECK (LENGTH(hash) = 32),
    "size" bigint NOT NULL,
    stripped_size bigint NOT NULL,
    weight integer NOT NULL,
    "height" integer UNIQUE NOT NULL CHECK (HEIGHT >= 0),
    "version" integer NOT NULL,
    hash_merkle_root bytea CHECK (LENGTH(hash_merkle_root) = 32) NOT NULL,
    -- witness_root bytea CHECK (LENGTH(witness_root) = 32) NOT NULL,
    -- tree_root bytea CHECK (LENGTH(tree_root) = 32) NOT NULL,
    -- reserved_root bytea CHECK (LENGTH(reserved_root) = 32) NOT NULL,
    -- mask bytea CHECK (LENGTH(mask) = 32) NOT NULL,
    "time" integer NOT NULL,
    median_time integer NOT NULL,
    nonce bigint NOT NULL,
    bits bytea CHECK (LENGTH(bits) = 4) NOT NULL,
    difficulty double precision NOT NULL,
    chainwork bytea CHECK (LENGTH(chainwork) = 32) NOT NULL,
    -- extra_nonce bytea CHECK (LENGTH(extra_nonce) = 24) NOT NULL,
    orphan boolean NOT NULL DEFAULT FALSE
);


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE blocks;
-- +goose StatementEnd
