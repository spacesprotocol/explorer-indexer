package sync

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/jinzhu/copier"
	"github.com/spacesprotocol/explorer-backend/pkg/db"
	"github.com/spacesprotocol/explorer-backend/pkg/node"
	. "github.com/spacesprotocol/explorer-backend/pkg/types"
)

type AggregateStats struct {
	StartHeight  int32
	EndHeight    int32
	TotalTime    time.Duration
	TotalBlocks  int
	TotalTx      int
	TotalInputs  int
	TotalOutputs int
	StartTime    time.Time
}

var (
	currentAggregation *AggregateStats
	aggregationSize    = 100 // Size of batch for aggregation
)

func logAggregateStats(stats *AggregateStats) {
	timePerBlock := float64(stats.TotalTime.Milliseconds()) / float64(stats.TotalBlocks)
	timePerTx := float64(stats.TotalTime.Milliseconds()) / float64(stats.TotalTx)
	timePerIO := float64(stats.TotalTime.Milliseconds()) / float64(stats.TotalInputs+stats.TotalOutputs)

	log.Printf("\n=== Aggregate Statistics for blocks %d-%d ===", stats.StartHeight, stats.EndHeight)
	log.Printf("Total time: %v", stats.TotalTime)
	log.Printf("Total blocks: %d (%.2f ms/block)", stats.TotalBlocks, timePerBlock)
	log.Printf("Total transactions: %d (%.2f ms/tx)", stats.TotalTx, timePerTx)
	log.Printf("Total I/O operations: %d (%.2f ms/op)",
		stats.TotalInputs+stats.TotalOutputs, timePerIO)
}

func updateAggregateStats(height int32, txCount, inputCount, outputCount int) {
	if currentAggregation == nil {
		currentAggregation = &AggregateStats{
			StartHeight: height,
			StartTime:   time.Now(),
		}
	}

	// Update counters
	currentAggregation.TotalBlocks++
	currentAggregation.TotalTx += txCount
	currentAggregation.TotalInputs += inputCount
	currentAggregation.TotalOutputs += outputCount

	// If we've reached the batch size, log and reset
	if currentAggregation.TotalBlocks >= aggregationSize {
		currentAggregation.EndHeight = height
		currentAggregation.TotalTime = time.Since(currentAggregation.StartTime)
		logAggregateStats(currentAggregation)
		currentAggregation = nil
	}
}

func SyncSpacesTransactions(txs []node.MetaTransaction, blockHash Bytes, sqlTx pgx.Tx) (pgx.Tx, error) {
	q := db.New(sqlTx)
	for _, tx := range txs {
		for _, create := range tx.Creates {
			vmet := db.InsertVMetaOutParams{
				BlockHash:    blockHash,
				Txid:         tx.TxID,
				Value:        pgtype.Int8{Int64: int64(create.Value), Valid: true},
				Scriptpubkey: &create.ScriptPubKey,
			}
			if create.Name != "" {
				if create.Name[0] == '@' {
					vmet.Name = pgtype.Text{
						String: create.Name[1:],
						Valid:  true,
					}
				} else {
					vmet.Name = pgtype.Text{
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
					vmet.BurnIncrement = pgtype.Int8{Int64: int64(*create.Covenant.BurnIncrement), Valid: true}
				}

				if create.Covenant.TotalBurned != nil {
					vmet.TotalBurned = pgtype.Int8{Int64: int64(*create.Covenant.TotalBurned), Valid: true}
				}

				if create.Covenant.ClaimHeight != nil {
					vmet.ClaimHeight = pgtype.Int8{Int64: int64(*create.Covenant.ClaimHeight), Valid: true}
				}

				if create.Covenant.ExpireHeight != nil {
					vmet.ExpireHeight = pgtype.Int8{Int64: int64(*create.Covenant.ExpireHeight), Valid: true}
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
				Value:        pgtype.Int8{Int64: int64(update.Output.Value), Valid: true},
				Scriptpubkey: &update.Output.ScriptPubKey,
			}

			if update.Priority != 0 {
				vmet.Priority = pgtype.Int8{Int64: int64(update.Priority), Valid: true}
			}

			if update.Reason != "" {
				vmet.Reason = pgtype.Text{String: update.Reason, Valid: true}
			}

			if update.Output.Name != "" {
				if update.Output.Name[0] == '@' {
					vmet.Name = pgtype.Text{
						String: update.Output.Name[1:],
						Valid:  true,
					}
				} else {
					vmet.Name = pgtype.Text{
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
				vmet.BurnIncrement = pgtype.Int8{
					Int64: int64(*covenant.BurnIncrement),
					Valid: true,
				}
			}

			if covenant.TotalBurned != nil {
				vmet.TotalBurned = pgtype.Int8{
					Int64: int64(*covenant.TotalBurned),
					Valid: true,
				}
			}

			if covenant.ClaimHeight != nil {
				vmet.ClaimHeight = pgtype.Int8{
					Int64: int64(*covenant.ClaimHeight),
					Valid: true,
				}
			}

			if covenant.ExpireHeight != nil {
				vmet.ExpireHeight = pgtype.Int8{
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
						vmet.Name = pgtype.Text{
							String: spend.ScriptError.Name[1:],
							Valid:  true,
						}
					} else {
						vmet.Name = pgtype.Text{
							String: spend.ScriptError.Name,
							Valid:  true,
						}
					}
				}

				if spend.ScriptError.Reason != "" {
					vmet.ScriptError = pgtype.Text{String: spend.ScriptError.Reason, Valid: true}
				}

				//TODO handle script error types gracefully
				if strings.ToUpper(spend.ScriptError.Type) == "REJECT" {
					vmet.Action = db.NullCovenantAction{CovenantAction: db.CovenantActionREJECT, Valid: true}
				} else {
					vmet.Action = db.NullCovenantAction{CovenantAction: db.CovenantActionREJECT, Valid: true}
					vmet.ScriptError = pgtype.Text{String: spend.ScriptError.Reason + string(spend.ScriptError.Type), Valid: true}
				}
			}

			if err := q.InsertVMetaOut(context.Background(), vmet); err != nil {
				return sqlTx, err
			}

		}
	}

	return sqlTx, nil
}

func SyncBlock(block *node.Block, tx pgx.Tx) (pgx.Tx, error) {
	q := db.New(tx)
	blockParams := db.InsertBlockParams{}
	copier.Copy(&blockParams, &block)
	if err := q.InsertBlock(context.Background(), blockParams); err != nil {
		return tx, err
	}
	txCount := 0
	inputCount := 0
	outputCount := 0
	for tx_index, transaction := range block.Transactions {
		ind := int32(tx_index)
		if err := insertTransaction(q, &transaction, &blockParams.Hash, &ind); err != nil {
			return tx, err
		}

		// Collect metrics
		txCount++
		inputCount += len(transaction.Vin)
		outputCount += len(transaction.Vout)
	}
	updateAggregateStats(block.Height, txCount, inputCount, outputCount)
	return tx, nil
}

func insertTransaction(q *db.Queries, transaction *node.Transaction, blockHash *Bytes, txIndex *int32) error {
	transactionParams := db.InsertTransactionParams{}
	copier.Copy(&transactionParams, &transaction)
	transactionParams.BlockHash = *blockHash
	var nullableIndex pgtype.Int4
	if txIndex == nil {
		nullableIndex.Valid = false
	} else {
		nullableIndex.Valid = true
		nullableIndex.Int32 = *txIndex
	}
	transactionParams.Index = nullableIndex

	// Insert the transaction first
	if err := q.InsertTransaction(context.Background(), transactionParams); err != nil {
		return err
	}

	// Prepare batch inputs
	inputs := make([]db.InsertBatchTxInputsParams, 0, len(transaction.Vin))
	var spenderUpdates []db.SetSpenderParams

	for input_index, txInput := range transaction.Vin {
		inputParam := db.InsertBatchTxInputsParams{
			BlockHash:    *blockHash,
			Txid:         transactionParams.Txid,
			Index:        int64(input_index),
			HashPrevout:  txInput.HashPrevout,
			IndexPrevout: int64(txInput.IndexPrevout),
			Sequence:     int64(txInput.Sequence),
			Coinbase:     txInput.Coinbase,
		}
		inputs = append(inputs, inputParam)

		if txInput.Coinbase == nil {
			var nullableIndex64 pgtype.Int8
			nullableIndex64.Valid = true
			nullableIndex64.Int64 = int64(input_index)
			spenderUpdates = append(spenderUpdates, db.SetSpenderParams{
				Txid:         *(txInput.HashPrevout),
				Index:        int64(txInput.IndexPrevout),
				SpenderTxid:  &transactionParams.Txid,
				SpenderIndex: nullableIndex64,
			})
		}
	}

	// Prepare batch outputs
	outputs := make([]db.InsertBatchTxOutputsParams, 0, len(transaction.Vout))
	for output_index, txOutput := range transaction.Vout {
		outputParam := db.InsertBatchTxOutputsParams{
			BlockHash:    *blockHash,
			Txid:         transactionParams.Txid,
			Index:        int64(output_index),
			Value:        int64(txOutput.Value()),
			Scriptpubkey: *txOutput.Scriptpubkey(),
		}
		outputs = append(outputs, outputParam)
	}

	// Batch insert inputs
	if len(inputs) > 0 {
		if _, err := q.InsertBatchTxInputs(context.Background(), inputs); err != nil {
			return fmt.Errorf("batch insert inputs: %w", err)
		}
	}

	// Batch insert outputs
	if len(outputs) > 0 {
		if _, err := q.InsertBatchTxOutputs(context.Background(), outputs); err != nil {
			return fmt.Errorf("batch insert outputs: %w", err)
		}
	}

	// Process spender updates
	for _, update := range spenderUpdates {
		if err := q.SetSpender(context.Background(), update); err != nil {
			return err
		}
	}

	return nil
}

func insertTransaction2(q *db.Queries, transaction *node.Transaction, blockHash *Bytes, txIndex *int32) error {
	transactionParams := db.InsertTransactionParams{}
	copier.Copy(&transactionParams, &transaction)
	var err error
	transactionParams.BlockHash = *blockHash
	var nullableIndex pgtype.Int4
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
			var nullableIndex64 pgtype.Int8
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
