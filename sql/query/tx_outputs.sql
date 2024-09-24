-- name: InsertTxOutput :exec
INSERT INTO tx_outputs (block_hash, txid, index, value, scriptPubKey)
VALUES ($1, $2, $3, $4, $5);

-- name: GetTxOutputsByTxid :many
SELECT *
FROM tx_outputs
WHERE txid = $1
ORDER BY index;

-- name: GetTxOutputsByBlockAndTxid :many
SELECT *
FROM tx_outputs
WHERE block_hash = $1 and txid = $2
ORDER BY index;
