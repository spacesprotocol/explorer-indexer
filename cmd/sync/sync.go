package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jinzhu/copier"
	"github.com/spacesprotocol/explorer-backend/pkg/db"
	"github.com/spacesprotocol/explorer-backend/pkg/node"
	. "github.com/spacesprotocol/explorer-backend/pkg/types"
)

func syncSpacesTransactions(txs []node.Transaction, blockHash Bytes, sqlTx *sql.Tx) (*sql.Tx, error) {
	q := db.New(sqlTx)
	for _, transaction := range txs {
		for vmetaout_index, vmetaout := range transaction.VMetaOut {
			//TODO throw an error, current behaviour is to skip
			if len(vmetaout.Outpoint) == 0 {
				// log.Printf("found bad vmetaout %+v skipping", vmetaout)
				// return sqlTx, fmt.Errorf("found bad vmetaout without outpoint %+v", vmetaout)
				continue
			}
			vmet := db.InsertVMetaOutParams{}
			copier.Copy(&vmet, &vmetaout)
			var nullableInt sql.NullInt64
			if vmetaout.Covenant.BurnIncrement == nil {
				nullableInt.Valid = false
			} else {
				nullableInt.Valid = true
				nullableInt.Int64 = int64(*vmetaout.Covenant.BurnIncrement)
			}
			vmet.BurnIncrement = nullableInt
			vmet.TxIndex = int64(vmetaout_index)
			vmet.Txid = transaction.Txid
			vmet.BlockHash = blockHash
			if len(vmetaout.ResponseName) > 0 && vmetaout.ResponseName[0] == '@' {
				vmet.Name = vmetaout.ResponseName[1:]
			} else {
				return sqlTx, fmt.Errorf("invalid spaces name %s", vmet.Name)
			}
			switch vmetaout.Covenant.Type {
			case "bid":
				vmet.CovenantAction = db.CovenantActionBID
			case "reserve":
				vmet.CovenantAction = db.CovenantActionRESERVE
			case "transfer":
				vmet.CovenantAction = db.CovenantActionTRANSFER
			default:
				return sqlTx, errors.New("Unknown covenant action")
			}
			if err := q.InsertVMetaOut(context.Background(), vmet); err != nil {
				return sqlTx, err
			}
		}
	}
	return sqlTx, nil
}

func syncBlock(block *node.Block, sqlTx *sql.Tx) (*sql.Tx, error) {
	q := db.New(sqlTx)
	blockParams := db.InsertBlockParams{}
	copier.Copy(&blockParams, &block)
	if err := q.InsertBlock(context.Background(), blockParams); err != nil {
		return sqlTx, err
	}
	for tx_index, transaction := range block.Transactions {
		ind := int32(tx_index)
		if err := insertTransaction(q, &transaction, &blockParams.Hash, &ind); err != nil {
			return sqlTx, err
		}
	}
	return sqlTx, nil
}

func insertTransaction(q *db.Queries, transaction *node.Transaction, blockHash *Bytes, txIndex *int32) error {
	transactionParams := db.InsertTransactionParams{}
	copier.Copy(&transactionParams, &transaction)
	var err error
	transactionParams.BlockHash = blockHash
	var nullableIndex sql.NullInt32
	if txIndex == nil {
		nullableIndex.Valid = false
	} else {
		nullableIndex.Valid = true
		nullableIndex.Int32 = *txIndex
	}
	transactionParams.Index = nullableIndex
	if err = q.InsertTransaction(context.Background(), transactionParams); err != nil {
		return err
	}
	for input_index, txInput := range transaction.Vin {
		txInputParams := db.InsertTxInputParams{}
		copier.Copy(&txInputParams, &txInput)
		txInputParams.BlockHash = *blockHash
		txInputParams.Txid = transactionParams.Txid
		txInputParams.Index = int64(input_index)

		if err := q.InsertTxInput(context.Background(), txInputParams); err != nil {
			return err
		}

		if txInputParams.Coinbase == nil {
			var nullableIndex64 sql.NullInt64
			nullableIndex64.Valid = true
			nullableIndex64.Int64 = int64(input_index)
			setSpenderParams := db.SetSpenderParams{
				// BlockHash:    txInputParams.BlockHash, do i need it?
				Txid:         *(txInputParams.HashPrevout),
				Index:        txInputParams.IndexPrevout,
				SpenderTxid:  &transactionParams.Txid,
				SpenderIndex: nullableIndex64,
			}
			if err = q.SetSpender(context.Background(), setSpenderParams); err != nil {
				return err
			}
		}
	}
	for output_index, txOutput := range transaction.Vout {
		txOutputParams := db.InsertTxOutputParams{}
		txOutputParams.Txid = transactionParams.Txid
		txOutputParams.BlockHash = *blockHash
		copier.Copy(&txOutputParams, &txOutput)
		txOutputParams.Index = int64(output_index)
		if err := q.InsertTxOutput(context.Background(), txOutputParams); err != nil {
			return err
		}
	}
	return nil
}
