-- +goose Up
-- +goose StatementBegin
ALTER TABLE blocks ADD COLUMN pointers_root bytea CHECK(pointers_root IS NULL OR length(pointers_root) = 32);
ALTER TABLE blocks RENAME COLUMN root_anchor TO spaces_root;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE blocks DROP COLUMN IF EXISTS pointers_root;
ALTER TABLE blocks RENAME COLUMN spaces_root TO root_anchor;
-- +goose StatementEnd
