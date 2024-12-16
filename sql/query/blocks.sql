-- name: UpsertBlock :one
INSERT INTO blocks (
    hash,
    size,
    stripped_size,
    weight,
    height,
    version,
    hash_merkle_root,
    time,
    median_time,
    nonce,
    bits,
    difficulty,
    chainwork
)
VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
ON CONFLICT (hash) DO UPDATE
SET
    height = EXCLUDED.height,
    orphan = false
RETURNING (xmax IS NOT NULL)::boolean AS was_updated;


-- name: GetBlocks :many
SELECT blocks.*, (
  SELECT COUNT(*) FROM transactions WHERE blocks.hash = transactions.block_hash
)::integer AS txs_count
FROM blocks
ORDER BY height DESC
LIMIT $1 OFFSET $2;

-- name: GetBlockByHash :one
SELECT blocks.*, (
  SELECT COUNT(*) FROM transactions WHERE blocks.hash = transactions.block_hash
)::integer AS txs_count
FROM blocks
WHERE blocks.hash = $1;

-- name: GetBlockByHeight :one
SELECT blocks.*, (
  SELECT COUNT(*) FROM transactions WHERE blocks.hash = transactions.block_hash
)::integer AS txs_count
FROM blocks
WHERE blocks.height = $1;

-- name: GetBlockHashByHeight :one
SELECT hash
FROM blocks
WHERE height = $1;

-- name: GetBlocksMaxHeight :one
SELECT COALESCE(MAX(height), -1)::integer
FROM blocks;

-- name: DeleteBlocksAfterHeight :exec
DELETE FROM blocks
WHERE height > $1;

-- name: SetOrphanAfterHeight :exec
UPDATE blocks SET orphan = true WHERE height > $1;

-- name: SetNegativeHeightToOrphans :exec
UPDATE blocks SET height = -2 WHERE orphan = true;
