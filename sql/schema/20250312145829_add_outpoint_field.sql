-- +goose Up
-- +goose StatementBegin
-- Add outpoint columns to vmetaouts table
ALTER TABLE public.vmetaouts
ADD COLUMN outpoint_txid bytea,
ADD COLUMN outpoint_index bigint;

CREATE INDEX index_vmetaouts_outpoint ON vmetaouts(outpoint_txid, outpoint_index)
WHERE outpoint_txid IS NOT NULL AND outpoint_index IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Drop index first
DROP INDEX IF EXISTS index_vmetaouts_outpoint;

-- Then drop columns
ALTER TABLE public.vmetaouts DROP COLUMN IF EXISTS outpoint_txid;
ALTER TABLE public.vmetaouts DROP COLUMN IF EXISTS outpoint_index;
-- +goose StatementEnd
