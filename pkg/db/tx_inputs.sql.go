// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: tx_inputs.sql

package db

import (
	"context"

	"github.com/spacesprotocol/explorer-backend/pkg/types"
)

const getTxInputsByBlockAndTxid = `-- name: GetTxInputsByBlockAndTxid :many
SELECT block_hash, txid, index, hash_prevout, index_prevout, sequence, coinbase, txinwitness
FROM tx_inputs
WHERE txid = $1 AND block_hash = $2
ORDER BY index
`

type GetTxInputsByBlockAndTxidParams struct {
	Txid      types.Bytes
	BlockHash types.Bytes
}

func (q *Queries) GetTxInputsByBlockAndTxid(ctx context.Context, arg GetTxInputsByBlockAndTxidParams) ([]TxInput, error) {
	rows, err := q.db.Query(ctx, getTxInputsByBlockAndTxid, arg.Txid, arg.BlockHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []TxInput{}
	for rows.Next() {
		var i TxInput
		if err := rows.Scan(
			&i.BlockHash,
			&i.Txid,
			&i.Index,
			&i.HashPrevout,
			&i.IndexPrevout,
			&i.Sequence,
			&i.Coinbase,
			&i.Txinwitness,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getTxInputsByTxid = `-- name: GetTxInputsByTxid :many
SELECT block_hash, txid, index, hash_prevout, index_prevout, sequence, coinbase, txinwitness
FROM tx_inputs
WHERE txid = $1
ORDER BY index
`

func (q *Queries) GetTxInputsByTxid(ctx context.Context, txid types.Bytes) ([]TxInput, error) {
	rows, err := q.db.Query(ctx, getTxInputsByTxid, txid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []TxInput{}
	for rows.Next() {
		var i TxInput
		if err := rows.Scan(
			&i.BlockHash,
			&i.Txid,
			&i.Index,
			&i.HashPrevout,
			&i.IndexPrevout,
			&i.Sequence,
			&i.Coinbase,
			&i.Txinwitness,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

type InsertBatchTxInputsParams struct {
	BlockHash    types.Bytes
	Txid         types.Bytes
	Index        int64
	HashPrevout  *types.Bytes
	IndexPrevout int64
	Sequence     int64
	Coinbase     *types.Bytes
	Txinwitness  []types.Bytes
}

const insertTxInput = `-- name: InsertTxInput :exec
INSERT INTO tx_inputs (block_hash, txid, index, hash_prevout, index_prevout, sequence, coinbase, txinwitness)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
`

type InsertTxInputParams struct {
	BlockHash    types.Bytes
	Txid         types.Bytes
	Index        int64
	HashPrevout  *types.Bytes
	IndexPrevout int64
	Sequence     int64
	Coinbase     *types.Bytes
	Txinwitness  []types.Bytes
}

func (q *Queries) InsertTxInput(ctx context.Context, arg InsertTxInputParams) error {
	_, err := q.db.Exec(ctx, insertTxInput,
		arg.BlockHash,
		arg.Txid,
		arg.Index,
		arg.HashPrevout,
		arg.IndexPrevout,
		arg.Sequence,
		arg.Coinbase,
		arg.Txinwitness,
	)
	return err
}
