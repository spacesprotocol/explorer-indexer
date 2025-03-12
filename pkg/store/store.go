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

// blockTimings collects timing statistics for block processing
type blockTimings struct {
	baseInsertTime time.Duration
	inputsTime     time.Duration
	outputsTime    time.Duration
	spendersTime   time.Duration
	spacesTime     time.Duration
	totalTxs       int
	totalInputs    int
	totalOutputs   int
	totalSpenders  int
	totalSpacesTxs int
}

func (bt *blockTimings) report(blockHeight int32) {
	log.Printf("Block %d timing summary:", blockHeight)
	log.Printf("  Total transactions: %d", bt.totalTxs)
	log.Printf("  Base tx insert time: %s (avg: %s per tx)",
		bt.baseInsertTime,
		time.Duration(int64(bt.baseInsertTime)/int64(max(1, bt.totalTxs))))
	log.Printf("  Inputs processing: %s for %d inputs (avg: %s per input)",
		bt.inputsTime,
		bt.totalInputs,
		time.Duration(int64(bt.inputsTime)/int64(max(1, bt.totalInputs))))
	log.Printf("  Outputs processing: %s for %d outputs (avg: %s per output)",
		bt.outputsTime,
		bt.totalOutputs,
		time.Duration(int64(bt.outputsTime)/int64(max(1, bt.totalOutputs))))
	log.Printf("  Spenders processing: %s for %d spenders (avg: %s per spender)",
		bt.spendersTime,
		bt.totalSpenders,
		time.Duration(int64(bt.spendersTime)/int64(max(1, bt.totalSpenders))))
	if bt.totalSpacesTxs > 0 {
		log.Printf("  Spaces transactions: %s for %d txs (avg: %s per tx)",
			bt.spacesTime,
			bt.totalSpacesTxs,
			time.Duration(int64(bt.spacesTime)/int64(bt.totalSpacesTxs)))
	}
	log.Printf("  Total processing time: %s",
		bt.baseInsertTime+bt.inputsTime+bt.outputsTime+bt.spendersTime+bt.spacesTime)
}

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

func StoreBitcoinBlock(block *node.Block, tx pgx.Tx) (pgx.Tx, error, *blockTimings) {
	timings := &blockTimings{}

	q := db.New(tx)
	blockParams := db.UpsertBlockParams{}
	copier.Copy(&blockParams, &block)

	wasInserted, err := q.UpsertBlock(context.Background(), blockParams)
	if err != nil {
		return tx, err, timings
	}

	if wasInserted {
		timings.totalTxs = len(block.Transactions)
		for tx_index, transaction := range block.Transactions {
			ind := int32(tx_index)
			if tx_index%50 == 0 {
				log.Print("current batch insert of tx # ", tx_index, " tx_hash ", transaction.Txid.String())
			}

			// Base transaction insert
			start := time.Now()
			err = storeTransactionBase(q, &transaction, &blockParams.Hash, &ind)
			timings.baseInsertTime += time.Since(start)
			if err != nil {
				return tx, err, timings
			}

			// Store inputs/outputs with timing
			start = time.Now()
			inCount, outCount, err := storeInputsOutputs(q, &transaction, &blockParams.Hash)
			ioTime := time.Since(start)
			if err != nil {
				return tx, err, timings
			}
			// Split the I/O time proportionally between inputs and outputs
			if total := inCount + outCount; total > 0 {
				timings.inputsTime += ioTime * time.Duration(inCount) / time.Duration(total)
				timings.outputsTime += ioTime * time.Duration(outCount) / time.Duration(total)
			}
			timings.totalInputs += inCount
			timings.totalOutputs += outCount

			// Update spenders
			start = time.Now()
			spendCount, err := UpdateTxSpenders(q, &transaction, blockParams.Hash)
			timings.spendersTime += time.Since(start)
			if err != nil {
				return tx, err, timings
			}
			timings.totalSpenders += spendCount
		}
	}

	return tx, nil, timings
}

func storeTransactionBase(q *db.Queries, transaction *node.Transaction, blockHash *Bytes, txIndex *int32) error {
	if blockHash.String() != deadbeefString {
		params := db.InsertTransactionParams{}
		copier.Copy(&params, transaction)
		params.BlockHash = *blockHash
		params.Index = *txIndex
		return q.InsertTransaction(context.Background(), params)
	}
	params := db.InsertMempoolTransactionParams{}
	copier.Copy(&params, transaction)
	params.BlockHash = *blockHash
	return q.InsertMempoolTransaction(context.Background(), params)
}

func storeInputsOutputs(q *db.Queries, transaction *node.Transaction, blockHash *Bytes) (inputCount, outputCount int, err error) {
	inputs := make([]db.InsertBatchTxInputsParams, 0, len(transaction.Vin))
	outputs := make([]db.InsertBatchTxOutputsParams, 0, len(transaction.Vout))

	// Prepare inputs
	for input_index, txInput := range transaction.Vin {
		var scriptSig Bytes
		if txInput.ScriptSig != nil {
			if err := scriptSig.UnmarshalText([]byte(txInput.ScriptSig.Hex)); err != nil {
				return 0, 0, fmt.Errorf("failed to unmarshal scriptsig: %w", err)
			}
		}
		inputParam := db.InsertBatchTxInputsParams{
			BlockHash:    *blockHash,
			Txid:         transaction.Txid,
			Index:        int64(input_index),
			HashPrevout:  txInput.HashPrevout,
			IndexPrevout: int64(txInput.IndexPrevout),
			Sequence:     int64(txInput.Sequence),
			Coinbase:     txInput.Coinbase,
			Txinwitness:  txInput.TxinWitness,
			Scriptsig:    &scriptSig,
		}
		inputs = append(inputs, inputParam)
	}

	// Prepare outputs
	for output_index, txOutput := range transaction.Vout {
		outputParam := db.InsertBatchTxOutputsParams{
			BlockHash:    *blockHash,
			Txid:         transaction.Txid,
			Index:        int64(output_index),
			Value:        int64(txOutput.Value()),
			Scriptpubkey: *txOutput.Scriptpubkey(),
		}
		outputs = append(outputs, outputParam)
	}

	if len(inputs) > 0 {
		if _, err := q.InsertBatchTxInputs(context.Background(), inputs); err != nil {
			return 0, 0, fmt.Errorf("batch insert inputs: %w", err)
		}
	}

	if len(outputs) > 0 {
		if _, err := q.InsertBatchTxOutputs(context.Background(), outputs); err != nil {
			return 0, 0, fmt.Errorf("batch insert outputs: %w", err)
		}
	}

	return len(inputs), len(outputs), nil
}

func UpdateTxSpenders(q *db.Queries, transaction *node.Transaction, blockHash Bytes) (int, error) {
	batchParams := db.SetSpenderBatchParams{
		Column1: make([]Bytes, 0, len(transaction.Vin)),
		Column2: make([]int64, 0, len(transaction.Vin)),
		Column3: make([]Bytes, 0, len(transaction.Vin)),
		Column4: make([]int64, 0, len(transaction.Vin)),
		Column5: make([]Bytes, 0, len(transaction.Vin)),
	}

	for input_index, txInput := range transaction.Vin {
		if txInput.Coinbase == nil {
			batchParams.Column1 = append(batchParams.Column1, *txInput.HashPrevout)
			batchParams.Column2 = append(batchParams.Column2, int64(txInput.IndexPrevout))
			batchParams.Column3 = append(batchParams.Column3, transaction.Txid)
			batchParams.Column4 = append(batchParams.Column4, int64(input_index))
			batchParams.Column5 = append(batchParams.Column5, blockHash)
		}
	}

	if len(batchParams.Column1) > 0 {
		if err := q.SetSpenderBatch(context.Background(), batchParams); err != nil {
			return 0, err
		}
	}

	return len(batchParams.Column1), nil
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

	// Store Bitcoin block and collect timings
	var timings *blockTimings
	// tx, err, timings = StoreBitcoinBlock(block, tx)
	if err != nil {
		return err
	}

	// Process Spaces data if applicable
	if block.Height >= activationBlock {
		// start := time.Now()
		spacesBlock, err := sc.GetBlockMeta(ctx, block.Hash.String())
		if err != nil {
			return err
		}

		tx, err = StoreSpacesTransactions(spacesBlock.Transactions, block.Hash, tx)
		if err != nil {
			return err
		}
		// timings.spacesTime = time.Since(start)
		// timings.totalSpacesTxs = len(spacesBlock.Transactions)
	}

	// Report aggregated timings before commit
	timings.report(block.Height)

	return tx.Commit(ctx)
}

func StoreTransaction(q *db.Queries, transaction *node.Transaction, blockHash *Bytes, txIndex *int32) error {
	if err := storeTransactionBase(q, transaction, blockHash, txIndex); err != nil {
		return err
	}

	if _, _, err := storeInputsOutputs(q, transaction, blockHash); err != nil {
		return err
	}

	if _, err := UpdateTxSpenders(q, transaction, *blockHash); err != nil {
		return err
	}

	return nil
}
