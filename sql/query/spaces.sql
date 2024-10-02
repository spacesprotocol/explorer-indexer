-- name: InsertVMetaOut :exec
INSERT INTO vmetaouts (block_hash, txid, tx_index, outpoint_txid, outpoint_index, name, burn_increment, covenant_action, claim_height, expire_height)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: GetVMetaOutsByTxid :many
SELECT *
FROM vmetaouts
WHERE txid = $1
ORDER BY tx_index;

-- name: GetVMetaoutsByBlockAndTxid :many
SELECT *
FROM vmetaouts
WHERE block_hash = $1 and txid = $2
ORDER BY tx_index;
