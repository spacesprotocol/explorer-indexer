-- name: InsertTransaction :exec
INSERT INTO transactions (
    txid, tx_hash, version, size, vsize, weight, locktime, fee, block_hash, index
) VALUES ( $1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: InsertMempoolTransaction :exec
INSERT INTO transactions (
    txid, tx_hash, version, size, vsize, weight, locktime, fee, block_hash
) VALUES ( $1, $2, $3, $4, $5, $6, $7, $8, $9);


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
WHERE block_hash = '\xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef';

-- name: DeleteMempoolTransactionByTxid :exec
DELETE FROM transactions
where txid = $1
AND block_hash = '\xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef';
