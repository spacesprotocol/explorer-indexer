-- +goose Up
-- +goose StatementBegin
INSERT INTO blocks (
    hash,
    size,
    stripped_size,
    weight,
    height,
    version,
    hash_merkle_root,
    time,
    median_time,
    nonce,
    bits,
    difficulty,
    chainwork
) VALUES (
    '\xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef'::bytea,
    0,  -- size
    0,  -- stripped_size
    0,  -- weight
    -1, -- special height for mempool
    1,  -- version
    '\xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef'::bytea,
    0,  -- time
    0,  -- median_time
    0,  -- nonce
    '\xdeadbeef'::bytea, -- bits
    0,  -- difficulty
    '\xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef'::bytea
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM blocks WHERE hash = '\xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef';
-- +goose StatementEnd
