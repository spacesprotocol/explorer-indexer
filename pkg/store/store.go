package store

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/jinzhu/copier"
	"github.com/spacesprotocol/explorer-indexer/pkg/db"
	"github.com/spacesprotocol/explorer-indexer/pkg/node"
	. "github.com/spacesprotocol/explorer-indexer/pkg/types"
)

const deadbeefString = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func StoreSpacesTransactions(txs []node.MetaTransaction, blockHash Bytes, sqlTx pgx.Tx) (pgx.Tx, error) {
	for _, tx := range txs {
		sqlTx, err := StoreSpacesTransaction(tx, blockHash, sqlTx)
		if err != nil {
			return sqlTx, err
		}
	}
	return sqlTx, nil
}

func StoreSpacesTransaction(tx node.MetaTransaction, blockHash Bytes, sqlTx pgx.Tx) (pgx.Tx, error) {
	q := db.New(sqlTx)
	for _, create := range tx.Creates {
		vmet := db.InsertVMetaOutParams{
			BlockHash:     blockHash,
			Txid:          tx.TxID,
			Value:         pgtype.Int8{Int64: int64(create.Value), Valid: true},
			Scriptpubkey:  &create.ScriptPubKey,
			OutpointTxid:  &tx.TxID,
			OutpointIndex: pgtype.Int8{Int64: int64(create.N), Valid: true},
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
			BlockHash:     blockHash,
			Txid:          tx.TxID,
			Value:         pgtype.Int8{Int64: int64(update.Output.Value), Valid: true},
			Scriptpubkey:  &update.Output.ScriptPubKey,
			OutpointTxid:  &update.Output.TxID,
			OutpointIndex: pgtype.Int8{Int64: int64(update.Output.N), Valid: true},
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

			if err := q.InsertVMetaOut(context.Background(), vmet); err != nil {
				return sqlTx, err
			}
		}

	}

	return sqlTx, nil
}

func StoreBitcoinBlock(block *node.Block, tx pgx.Tx) (pgx.Tx, error) {
	q := db.New(tx)
	blockParams := db.UpsertBlockParams{}
	copier.Copy(&blockParams, &block)
	wasInserted, err := q.UpsertBlock(context.Background(), blockParams)
	if err != nil {
		return tx, err
	}
	if wasInserted {
		// Prepare all transactions for batch insert
		batchParams := prepareBatchTransactions(block.Transactions, &blockParams.Hash)

		log.Printf("Batch inserting %d transactions for block", len(batchParams))

		// Batch insert all transactions at once using PostgreSQL COPY protocol
		rowsAffected, err := q.InsertBatchTransactions(context.Background(), batchParams)
		if err != nil {
			return tx, fmt.Errorf("batch insert transactions: %w", err)
		}

		log.Printf("Successfully inserted %d transactions", rowsAffected)
	}
	return tx, nil
}

func storeTransactionBase(q *db.Queries, transaction *node.Transaction, blockHash *Bytes, txIndex *int32) error {
	// Calculate aggregates for all transactions
	inputCount, outputCount, totalOutputValue := calculateAggregates(transaction)

	if blockHash.String() != deadbeefString {
		params := db.InsertTransactionParams{}
		copier.Copy(&params, transaction)
		params.BlockHash = *blockHash
		params.Index = *txIndex
		params.InputCount = inputCount
		params.OutputCount = outputCount
		params.TotalOutputValue = totalOutputValue
		return q.InsertTransaction(context.Background(), params)
	}
	params := db.InsertMempoolTransactionParams{}
	copier.Copy(&params, transaction)
	params.BlockHash = *blockHash
	params.InputCount = inputCount
	params.OutputCount = outputCount
	params.TotalOutputValue = totalOutputValue
	return q.InsertMempoolTransaction(context.Background(), params)
}

// calculateAggregates computes input/output counts and total output value
// Returns: inputCount, outputCount, totalOutputValue
func calculateAggregates(transaction *node.Transaction) (int32, int32, int64) {
	inputCount := int32(len(transaction.Vin))
	outputCount := int32(len(transaction.Vout))

	var totalOutputValue int64
	for _, txOutput := range transaction.Vout {
		totalOutputValue += int64(txOutput.Value())
	}

	return inputCount, outputCount, totalOutputValue
}

// prepareBatchTransactions prepares all transactions in a block for batch insertion
func prepareBatchTransactions(transactions []node.Transaction, blockHash *Bytes) []db.InsertBatchTransactionsParams {
	batch := make([]db.InsertBatchTransactionsParams, 0, len(transactions))

	for tx_index, transaction := range transactions {
		inputCount, outputCount, totalOutputValue := calculateAggregates(&transaction)

		params := db.InsertBatchTransactionsParams{
			Txid:             transaction.Txid,
			TxHash:           transaction.TxHash(),
			Version:          int32(transaction.Version),
			Size:             int64(transaction.Size),
			Vsize:            int64(transaction.VSize),
			Weight:           int64(transaction.Weight),
			Locktime:         int32(transaction.LockTime),
			Fee:              int64(transaction.Fee()),
			BlockHash:        *blockHash,
			Index:            int32(tx_index),
			InputCount:       inputCount,
			OutputCount:      outputCount,
			TotalOutputValue: totalOutputValue,
		}

		batch = append(batch, params)
	}

	return batch
}

// detects chain split (reorganization) and
// returns the height and blockhash of the last block that is identical in the db and in the node
func GetSyncedHead(pg *pgx.Conn, bc *node.BitcoinClient) (int32, *Bytes, error) {
	q := db.New(pg)
	//takes last block from the DB
	height, err := q.GetBlocksMaxHeight(context.Background())
	if err != nil {
		return -1, nil, err
	}
	//height is the height of the db block
	for height >= 0 {
		//take last block hash from the DB
		dbHash, err := q.GetBlockHashByHeight(context.Background(), height)
		if err != nil {
			return -1, nil, err
		}
		//takes the block of same height from the bitcoin node
		nodeHash, err := bc.GetBlockHash(context.Background(), int(height))
		if err != nil {
			return -1, nil, err
		}
		// nodeHash *bytes
		// dbHash Bytes
		if bytes.Equal(dbHash, *nodeHash) {
			//marking all the blocks in the DB after the sycned height as orphans
			if err := q.SetOrphanAfterHeight(context.Background(), height); err != nil {
				return -1, nil, err
			}
			if err := q.SetNegativeHeightToOrphans(context.Background()); err != nil {
				return -1, nil, err
			}
			return height, &dbHash, nil
		}
		height -= 1
	}
	return -1, nil, nil
}

func StoreBlock(ctx context.Context, pg *pgx.Conn, block *node.Block, sc *node.SpacesClient, activationBlock int32) error {
	totalStart := time.Now()
	defer func() {
		log.Printf("Total block %d processing time: %s", block.Height, time.Since(totalStart))
	}()

	log.Printf("trying to store block #%d", block.Height)

	tx, err := pg.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Store Bitcoin block
	tx, err = StoreBitcoinBlock(block, tx)
	if err != nil {
		return err
	}

	if block.Height >= activationBlock {
		spacesBlock, err := sc.GetBlockMeta(ctx, block.Hash.String())
		if err != nil {
			return err
		}

		tx, err = StoreSpacesTransactions(spacesBlock.Transactions, block.Hash, tx)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func StoreTransaction(q *db.Queries, transaction *node.Transaction, blockHash *Bytes, txIndex *int32) error {
	if err := storeTransactionBase(q, transaction, blockHash, txIndex); err != nil {
		return err
	}
	return nil
}
