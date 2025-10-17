-- name: InsertTransaction :exec
INSERT INTO transactions (
    txid, tx_hash, version, size, vsize, weight, locktime, fee, block_hash, index,
    input_count, output_count, total_output_value
) VALUES ( $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: InsertMempoolTransaction :exec
INSERT INTO transactions (
    txid, tx_hash, version, size, vsize, weight, locktime, fee, block_hash,
    input_count, output_count, total_output_value
) VALUES ( $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);

-- name: InsertBatchTransactions :copyfrom
INSERT INTO transactions (
    txid, tx_hash, version, size, vsize, weight, locktime, fee, block_hash, index,
    input_count, output_count, total_output_value
) VALUES ( $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

--- name: GetTransactionByTxid :one
SELECT
  transactions.*,
  COALESCE(blocks.height, -1)::integer AS block_height_not_null
FROM transactions
  LEFT JOIN blocks ON (transactions.block_hash = blocks.hash)
WHERE transactions.txid = $1;

-- name: GetTransactionsByBlockHeight :many
SELECT
  transactions.*,
  COALESCE(blocks.height, -1)::integer AS block_height_not_null
FROM
  transactions
  INNER JOIN blocks ON (transactions.block_hash = blocks.hash)
WHERE blocks.height = $1
ORDER BY transactions.index
LIMIT $2 OFFSET $3;


-- name: GetMempoolTxids :many
SELECT txid
FROM transactions
WHERE block_hash = '\xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef'
ORDER BY index;

-- name: GetMempoolTransactions :many
SELECT *
FROM transactions
WHERE block_hash = '\xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef'
ORDER BY index
LIMIT $1 OFFSET $2;

-- name: DeleteMempoolTransactions :exec
DELETE FROM transactions
WHERE block_hash = '\xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef' and index <0;

-- name: DeleteMempoolTransactionByTxid :exec
DELETE FROM transactions
where txid = $1
AND block_hash = '\xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef';

-- name: DeleteMempoolTransactionsByTxids :exec
DELETE FROM transactions
WHERE txid = ANY($1::bytea[])
AND block_hash = '\xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef';
