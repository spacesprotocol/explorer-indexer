-- name: InsertTxOutput :exec
INSERT INTO tx_outputs (block_hash, txid, index, value, scriptPubKey)
VALUES ($1, $2, $3, $4, $5);

-- name: InsertBatchTxOutputs :copyfrom
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


-- name: SetSpender :exec
UPDATE tx_outputs
SET spender_block_hash = $5, spender_txid = $3, spender_index = $4
WHERE txid = $1 AND index = $2;

-- name: DeleteMempoolTxOutputs :exec
DELETE FROM tx_outputs
WHERE block_hash = '\xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef';


-- name: SetSpenderBatch :exec
UPDATE tx_outputs
SET spender_block_hash = u.spender_block_hash,
    spender_txid = u.spender_txid,
    spender_index = u.spender_index
FROM (
    SELECT UNNEST($1::bytea[]) as txid,
           UNNEST($2::bigint[]) as index,
           UNNEST($3::bytea[]) as spender_txid,
           UNNEST($4::bigint[]) as spender_index,
           UNNEST($5::bytea[]) as spender_block_hash
) as u
WHERE tx_outputs.txid = u.txid
AND tx_outputs.index = u.index;
