package main

import (
	"context"
	"database/sql"

	"github.com/jinzhu/copier"
	"github.com/spacesprotocol/explorer-backend/pkg/db"
	"github.com/spacesprotocol/explorer-backend/pkg/node"
	. "github.com/spacesprotocol/explorer-backend/pkg/types"
)

func syncBlock(pg *sql.DB, block *node.Block) error {
	tx, err := pg.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	q := db.New(tx)
	blockParams := db.InsertBlockParams{}
	copier.Copy(&blockParams, &block)
	if err := q.InsertBlock(context.Background(), blockParams); err != nil {
		return err
	}
	for tx_index, transaction := range block.Transactions {
		ind := int32(tx_index)
		if err := insertTransaction(q, &transaction, &blockParams.Hash, &ind); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func insertTransaction(q *db.Queries, transaction *node.Transaction, blockHash *Bytes, index *int32) error {
	transactionParams := db.InsertTransactionParams{}
	copier.Copy(&transactionParams, &transaction)
	var err error
	transactionParams.BlockHash = blockHash
	var nullableIndex sql.NullInt32
	if index == nil {
		nullableIndex.Valid = false
	} else {
		nullableIndex.Valid = true
		nullableIndex.Int32 = *index
	}
	transactionParams.Index = nullableIndex
	if err = q.InsertTransaction(context.Background(), transactionParams); err != nil {
		return err
	}
	for index, txInput := range transaction.Vin {
		txInputParams := db.InsertTxInputParams{}
		txInputParams.BlockHash = *blockHash
		txInputParams.Txid = transactionParams.Txid
		txInputParams.Index = int64(index)
		copier.Copy(&txInputParams, &txInput)
		if err := q.InsertTxInput(context.Background(), txInputParams); err != nil {
			return err
		}
	}
	for index, txOutput := range transaction.Vout {
		txOutputParams := db.InsertTxOutputParams{}
		txOutputParams.Txid = transactionParams.Txid
		txOutputParams.BlockHash = *blockHash
		copier.Copy(&txOutputParams, &txOutput)
		txOutputParams.Index = int32(index)
		if err := q.InsertTxOutput(context.Background(), txOutputParams); err != nil {
			return err
		}
	}
	return nil
}
