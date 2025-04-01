-- +goose Up
-- +goose StatementBegin
-- Add outpoint columns to vmetaouts table
ALTER TABLE blocks
ADD COLUMN root_anchor bytea CHECK(root_anchor IS NULL OR length(root_anchor) = 32);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE blocks DROP COLUMN IF EXISTS root_anchor;
-- +goose StatementEnd
