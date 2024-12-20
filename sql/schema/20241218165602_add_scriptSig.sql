-- +goose Up
-- +goose StatementBegin
-- Add the column (fast metadata operation)
ALTER TABLE tx_inputs 
ADD COLUMN scriptSig bytea;

ALTER TABLE tx_inputs
ADD CONSTRAINT witness_or_scriptsig_or_coinbase 
CHECK (
    (coinbase IS NOT NULL) OR 
    (scriptSig IS NOT NULL) OR 
    (txinwitness IS NOT NULL AND array_length(txinwitness, 1) > 0)
) NOT VALID; -- we don't enforce it for already existing rows
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tx_inputs
DROP CONSTRAINT witness_or_scriptsig_or_coinbase;

ALTER TABLE tx_inputs
DROP COLUMN scriptSig;
-- +goose StatementEnd
