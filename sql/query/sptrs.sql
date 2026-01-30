-- name: InsertSpacePointer :exec
INSERT INTO space_pointers (
    block_hash,
    txid,
    vout,
    sptr,
    value,
    script_pubkey,
    data
)
VALUES ($1, $2, $3, $4, $5, $6, $7);


-- name: UpdateSpacePointerSpender :exec
UPDATE space_pointers
SET spent_block_hash = $1,
    spent_txid = $2,
    spent_vin = $3
WHERE block_hash = $4
  AND txid = $5
  AND vout = $6;


-- name: InsertDelegation :exec
INSERT INTO sptr_delegations (
    sptr,
    name,
    block_hash,
    txid,
    vout
)
VALUES ($1, $2, $3, $4, $5);


-- name: UpdateDelegationRevoked :exec
UPDATE sptr_delegations
SET revoked = true,
    revoked_block_hash = $1,
    revoked_txid = $2,
    revoked_vout = $3
WHERE block_hash = $4
  AND txid = $5
  AND vout = $6;


-- name: FindLatestDelegationBySptr :one
SELECT sptr_delegations.*
FROM sptr_delegations
JOIN blocks ON sptr_delegations.block_hash = blocks.hash
WHERE sptr_delegations.sptr = $1
  AND sptr_delegations.name = $2
  AND sptr_delegations.revoked = false
  AND blocks.orphan = false
ORDER BY blocks.height DESC, sptr_delegations.identifier DESC
LIMIT 1;


-- name: FindSpacePointerByOutpoint :one
SELECT space_pointers.*
FROM space_pointers
JOIN blocks ON space_pointers.block_hash = blocks.hash
WHERE space_pointers.txid = $1
  AND space_pointers.vout = $2
  AND blocks.orphan = false;


-- name: UpsertSpacePointer :exec
INSERT INTO space_pointers (
    block_hash,
    txid,
    vout,
    sptr,
    value,
    script_pubkey,
    data
)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (block_hash, txid, vout) DO NOTHING;


-- name: UpsertDelegation :exec
INSERT INTO sptr_delegations (
    sptr,
    name,
    block_hash,
    txid,
    vout
)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (block_hash, txid, vout) DO NOTHING;
