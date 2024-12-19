-- +goose Up
-- +goose StatementBegin
ALTER TABLE tx_inputs 
ADD COLUMN scriptSig bytea;

UPDATE tx_inputs 
SET scriptSig = '\xff'
WHERE NOT (
    (coinbase IS NOT NULL) OR 
    (txinwitness IS NOT NULL AND array_length(txinwitness, 1) > 0)
);

ALTER TABLE tx_inputs
ADD CONSTRAINT witness_or_scriptsig_or_coinbase 
CHECK (
    (coinbase IS NOT NULL) OR 
    (scriptSig IS NOT NULL and length(scriptSig) > 0) OR 
    (txinwitness IS NOT NULL AND array_length(txinwitness, 1) > 0)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tx_inputs
DROP CONSTRAINT witness_or_scriptsig_or_coinbase;

ALTER TABLE tx_inputs
DROP COLUMN scriptSig;
-- +goose StatementEnd
