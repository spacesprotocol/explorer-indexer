package store

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/jinzhu/copier"
	"github.com/spacesprotocol/explorer-backend/pkg/db"
	"github.com/spacesprotocol/explorer-backend/pkg/node"
	. "github.com/spacesprotocol/explorer-backend/pkg/types"
)

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
		for tx_index, transaction := range block.Transactions {
			ind := int64(tx_index)
			if err := StoreTransaction(q, &transaction, &blockParams.Hash, &ind); err != nil {
				return tx, err
			}
		}
	}
	return tx, nil
}

func UpdateBlockSpender(block *node.Block, tx pgx.Tx) (pgx.Tx, error) {
	log.Printf("updating spenders from the block %d", block.Height)
	q := db.New(tx)
	for _, transaction := range block.Transactions {
		if err := updateTxSpenders(q, &transaction, block.Hash); err != nil {
			return tx, err
		}
	}
	return tx, nil
}

func updateTxSpenders(q *db.Queries, transaction *node.Transaction, blockHash Bytes) error {
	batchParams := db.SetSpenderBatchParams{
		Column1: make([]Bytes, 0, len(transaction.Vin)), // txids
		Column2: make([]int64, 0, len(transaction.Vin)), // indices
		Column3: make([]Bytes, 0, len(transaction.Vin)), // spender_txids
		Column4: make([]int64, 0, len(transaction.Vin)), // spender_indices
		Column5: make([]Bytes, 0, len(transaction.Vin)), // spender_block_hashes
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
			return err
		}
	}

	return nil
}

func StoreTransaction(q *db.Queries, transaction *node.Transaction, blockHash *Bytes, txIndex *int64) error {
	transactionParams := db.InsertTransactionParams{}
	copier.Copy(&transactionParams, &transaction)
	transactionParams.BlockHash = *blockHash
	var nullableIndex pgtype.Int8
	if txIndex == nil {
		nullableIndex.Valid = false
	} else {
		nullableIndex.Valid = true
		nullableIndex.Int64 = *txIndex
	}
	transactionParams.Index = nullableIndex

	if err := q.InsertTransaction(context.Background(), transactionParams); err != nil {
		return err
	}

	inputs := make([]db.InsertBatchTxInputsParams, 0, len(transaction.Vin))

	for input_index, txInput := range transaction.Vin {

		var scriptSig Bytes
		if txInput.ScriptSig != nil {
			if err := scriptSig.UnmarshalText([]byte(txInput.ScriptSig.Hex)); err != nil {
				return fmt.Errorf("failed to unmarshal scriptsig: %w", err)
			}
		}
		inputParam := db.InsertBatchTxInputsParams{
			BlockHash:    *blockHash,
			Txid:         transactionParams.Txid,
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

	if len(inputs) > 0 {
		if _, err := q.InsertBatchTxInputs(context.Background(), inputs); err != nil {
			return fmt.Errorf("batch insert inputs: %w", err)
		}
	}

	if len(outputs) > 0 {
		if _, err := q.InsertBatchTxOutputs(context.Background(), outputs); err != nil {
			return fmt.Errorf("batch insert outputs: %w", err)
		}
	}

	if err := updateTxSpenders(q, transaction, *blockHash); err != nil {
		return err

	}

	// for _, update := range spenderUpdates {
	// 	if err := q.SetSpender(context.Background(), update); err != nil {
	// 		return err
	// 	}
	// }

	return nil
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
	log.Printf("trying to store block #%d", block.Height)
	tx, err := pg.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

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
