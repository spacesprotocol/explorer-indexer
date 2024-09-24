-- name: InsertTxInput :exec
INSERT INTO tx_inputs (block_hash, txid, index, hash_prevout, index_prevout, sequence, coinbase, txinwitness)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);


-- name: GetTxInputsByTxid :many
SELECT *
FROM tx_inputs
WHERE txid = $1
ORDER BY index;

-- name: GetTxInputsByBlockAndTxid :many
SELECT *
FROM tx_inputs
WHERE txid = $1 AND block_hash = $2
ORDER BY index;
