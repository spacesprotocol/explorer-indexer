package sync

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jinzhu/copier"
	"github.com/spacesprotocol/explorer-backend/pkg/db"
	"github.com/spacesprotocol/explorer-backend/pkg/node"
	. "github.com/spacesprotocol/explorer-backend/pkg/types"
)

func SyncSpacesTransactions(txs []node.MetaTransaction, blockHash Bytes, sqlTx *sql.Tx) (*sql.Tx, error) {
	q := db.New(sqlTx)
	for _, tx := range txs {
		for _, create := range tx.Creates {
			vmet := db.InsertVMetaOutParams{
				BlockHash:    blockHash,
				Txid:         tx.TxID,
				Value:        sql.NullInt64{int64(create.Value), true},
				Scriptpubkey: &create.ScriptPubKey,
			}
			if create.Name != "" {
				if create.Name[0] == '@' {
					vmet.Name = sql.NullString{
						String: create.Name[1:],
						Valid:  true,
					}
				} else {
					vmet.Name = sql.NullString{
						String: create.Name,
						Valid:  true,
					}
				}
			}

			if create.Covenant.Type != "" {
				switch strings.ToUpper(create.Covenant.Type) {
				case "BID":
					vmet.Action = db.NullCovenantAction{
						CovenantAction: db.CovenantActionBID,
						Valid:          true,
					}
				case "RESERVE":
					vmet.Action = db.NullCovenantAction{
						CovenantAction: db.CovenantActionRESERVE,
						Valid:          true,
					}
				case "TRANSFER":
					vmet.Action = db.NullCovenantAction{
						CovenantAction: db.CovenantActionTRANSFER,
						Valid:          true,
					}
				case "ROLLOUT":
					vmet.Action = db.NullCovenantAction{
						CovenantAction: db.CovenantActionROLLOUT,
						Valid:          true,
					}
				case "REVOKE":
					vmet.Action = db.NullCovenantAction{
						CovenantAction: db.CovenantActionREVOKE,
						Valid:          true,
					}
				default:
					return sqlTx, fmt.Errorf("unknown covenant action: %s", create.Covenant.Type)
				}

				if create.Covenant.BurnIncrement != nil {
					vmet.BurnIncrement = sql.NullInt64{Int64: int64(*create.Covenant.BurnIncrement), Valid: true}
				}

				if create.Covenant.TotalBurned != nil {
					vmet.TotalBurned = sql.NullInt64{Int64: int64(*create.Covenant.TotalBurned), Valid: true}
				}

				if create.Covenant.ClaimHeight != nil {
					vmet.ClaimHeight = sql.NullInt64{Int64: int64(*create.Covenant.ClaimHeight), Valid: true}
				}

				if create.Covenant.ExpireHeight != nil {
					vmet.ExpireHeight = sql.NullInt64{Int64: int64(*create.Covenant.ExpireHeight), Valid: true}
				}

				if create.Covenant.Signature != nil {
					vmet.Signature = &create.Covenant.Signature
				}
			}

			if err := q.InsertVMetaOut(context.Background(), vmet); err != nil {
				return sqlTx, err
			}
		}

		for _, update := range tx.Updates {
			vmet := db.InsertVMetaOutParams{
				BlockHash:    blockHash,
				Txid:         tx.TxID,
				Value:        sql.NullInt64{int64(update.Output.Value), true},
				Scriptpubkey: &update.Output.ScriptPubKey,
			}

			if update.Priority != 0 {
				vmet.Priority = sql.NullInt64{Int64: int64(update.Priority), Valid: true}
			}

			if update.Reason != "" {
				vmet.Reason = sql.NullString{update.Reason, true}
			}

			if update.Output.Name != "" {
				if update.Output.Name[0] == '@' {
					vmet.Name = sql.NullString{
						String: update.Output.Name[1:],
						Valid:  true,
					}
				} else {
					vmet.Name = sql.NullString{
						String: update.Output.Name,
						Valid:  true,
					}
				}
			}
			switch strings.ToUpper(update.Type) {
			case "BID":
				vmet.Action = db.NullCovenantAction{
					CovenantAction: db.CovenantActionBID,
					Valid:          true,
				}
			case "RESERVE":
				vmet.Action = db.NullCovenantAction{
					CovenantAction: db.CovenantActionRESERVE,
					Valid:          true,
				}
			case "TRANSFER":
				vmet.Action = db.NullCovenantAction{
					CovenantAction: db.CovenantActionTRANSFER,
					Valid:          true,
				}
			case "ROLLOUT":
				vmet.Action = db.NullCovenantAction{
					CovenantAction: db.CovenantActionROLLOUT,
					Valid:          true,
				}
			case "REVOKE":
				vmet.Action = db.NullCovenantAction{
					CovenantAction: db.CovenantActionREVOKE,
					Valid:          true,
				}
			default:
				return sqlTx, fmt.Errorf("unknown covenant action: %s", update.Type)
			}
			covenant := update.Output.Covenant
			if covenant.BurnIncrement != nil {
				vmet.BurnIncrement = sql.NullInt64{
					Int64: int64(*covenant.BurnIncrement),
					Valid: true,
				}
			}

			if covenant.TotalBurned != nil {
				vmet.TotalBurned = sql.NullInt64{
					Int64: int64(*covenant.TotalBurned),
					Valid: true,
				}
			}

			if covenant.ClaimHeight != nil {
				vmet.ClaimHeight = sql.NullInt64{
					Int64: int64(*covenant.ClaimHeight),
					Valid: true,
				}
			}

			if covenant.ExpireHeight != nil {
				vmet.ExpireHeight = sql.NullInt64{
					Int64: int64(*covenant.ExpireHeight),
					Valid: true,
				}
			}

			if covenant.Signature != nil {
				vmet.Signature = &covenant.Signature
			}

			if err := q.InsertVMetaOut(context.Background(), vmet); err != nil {
				return sqlTx, err
			}

		}

		for _, spend := range tx.Spends {
			vmet := db.InsertVMetaOutParams{
				BlockHash: blockHash,
				Txid:      tx.TxID,
			}

			if spend.ScriptError != nil {
				if spend.ScriptError.Name != "" {
					if spend.ScriptError.Name[0] == '@' {
						vmet.Name = sql.NullString{
							String: spend.ScriptError.Name[1:],
							Valid:  true,
						}
					} else {
						vmet.Name = sql.NullString{
							String: spend.ScriptError.Name,
							Valid:  true,
						}
					}
				}

				if spend.ScriptError.Reason != "" {
					vmet.ScriptError = sql.NullString{String: spend.ScriptError.Reason, Valid: true}
				}

				//TODO handle script error types gracefully
				if strings.ToUpper(spend.ScriptError.Type) == "REJECT" {
					vmet.Action = db.NullCovenantAction{CovenantAction: db.CovenantActionREJECT, Valid: true}
				} else {
					vmet.Action = db.NullCovenantAction{CovenantAction: db.CovenantActionREJECT, Valid: true}
					vmet.ScriptError = sql.NullString{String: spend.ScriptError.Reason + string(spend.ScriptError.Type), Valid: true}
				}
			}

			if err := q.InsertVMetaOut(context.Background(), vmet); err != nil {
				return sqlTx, err
			}

		}
	}

	return sqlTx, nil
}

func SyncBlock(block *node.Block, sqlTx *sql.Tx) (*sql.Tx, error) {
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
	transactionParams.BlockHash = *blockHash
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
