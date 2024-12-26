-- name: InsertVMetaOut :exec
INSERT INTO vmetaouts (
    block_hash,
    txid,
    priority,
    name,
    value,
    scriptPubKey,
    action,
    burn_increment,
    signature,
    total_burned,
    claim_height,
    expire_height,
    script_error,
    reason
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14);


-- name: InsertRollout :exec
INSERT INTO rollouts (
    name,
    bid,
    target
)
VALUES ($1, $2, $3);


-- name: DeleteRollouts :exec
DELETE FROM rollouts; 


-- name: DeleteMempoolVmetaouts :exec
DELETE FROM vmetaouts
WHERE block_hash = '\xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef';


-- name: DeleteMempoolVmetaoutsByTxid :exec
DELETE FROM vmetaouts
WHERE txid = $1
AND block_hash = '\xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef';
