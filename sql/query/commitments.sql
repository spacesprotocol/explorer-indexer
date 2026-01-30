-- name: InsertCommitment :exec
INSERT INTO commitments (block_hash, txid, name, state_root, history_hash, revocation) VALUES ($1, $2, $3, $4, $5, $6);

-- name: CommitmentExists :one
SELECT EXISTS(SELECT 1 FROM commitments WHERE name = $1 AND state_root = $2);

-- name: GetLatestCommitmentByName :one
SELECT commitments.* FROM commitments
JOIN blocks ON commitments.block_hash = blocks.hash
WHERE commitments.name = $1
ORDER BY blocks.height DESC
LIMIT 1;

-- name: GetCommitmentByBlockHashAndName :one
SELECT * FROM commitments WHERE block_hash = $1 AND name = $2;

-- name: GetCommitmentsByBlockHeightAndName :many
SELECT commitments.* FROM commitments
JOIN blocks ON commitments.block_hash = blocks.hash
WHERE blocks.height = $1 AND commitments.name = $2;
