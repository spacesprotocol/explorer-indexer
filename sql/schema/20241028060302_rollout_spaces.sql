-- +goose Up
-- +goose StatementBegin
CREATE TABLE rollouts (
    name TEXT NOT NULL CHECK (LENGTH(name) < 64),
    bid bigint,
    height bigint
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP table rollouts;
-- +goose StatementEnd

