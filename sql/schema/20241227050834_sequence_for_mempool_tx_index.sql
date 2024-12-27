-- +goose Up
-- +goose StatementBegin
CREATE SEQUENCE IF NOT EXISTS mempool_tx_index_seq
    INCREMENT BY -1
    MAXVALUE -1
    NO MINVALUE
    NO CYCLE;

ALTER TABLE transactions 
    ALTER COLUMN index SET DEFAULT nextval('mempool_tx_index_seq');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE transactions 
    ALTER COLUMN index DROP DEFAULT;

DROP SEQUENCE IF EXISTS mempool_tx_index_seq;
-- +goose StatementEnd
