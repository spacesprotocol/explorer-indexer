-- +goose Up
-- +goose StatementBegin
ALTER TABLE transactions ALTER COLUMN index TYPE bigint;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM transactions 
WHERE block_hash = '\xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef';

ALTER TABLE transactions ALTER COLUMN index TYPE integer;
-- +goose StatementEnd
