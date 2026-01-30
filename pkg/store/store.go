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

func StoreSpacesTransactions(ctx context.Context, txs []node.MetaTransaction, blockHash Bytes, sqlTx pgx.Tx) (pgx.Tx, error) {
	for _, tx := range txs {
		sqlTx, err := StoreSpacesTransaction(ctx, tx, blockHash, sqlTx)
		if err != nil {
			return sqlTx, err
		}
	}
	return sqlTx, nil
}

func StoreSpacesPtrTransactions(ctx context.Context, ptrTxs []node.PtrTxMeta, btcTxs []node.Transaction, blockHash Bytes, sqlTx pgx.Tx) (pgx.Tx, error) {
	// Create a map of txid -> Bitcoin transaction for quick lookup
	btcTxMap := make(map[string]*node.Transaction)
	for i := range btcTxs {
		btcTxMap[btcTxs[i].Txid.String()] = &btcTxs[i]
	}

	for _, ptrTx := range ptrTxs {
		btcTx := btcTxMap[ptrTx.TxID.String()]
		sqlTx, err := StoreSpacesPtrTransaction(ctx, ptrTx, btcTx, blockHash, sqlTx)
		if err != nil {
			return sqlTx, err
		}
	}
	return sqlTx, nil
}

func StoreSpacesPtrTransaction(ctx context.Context, tx node.PtrTxMeta, btcTx *node.Transaction, blockHash Bytes, sqlTx pgx.Tx) (pgx.Tx, error) {
	q := db.New(sqlTx)

	// Update spenders for space pointers being spent
	if btcTx != nil {
		for _, spendIndex := range tx.Spends {
			if int(spendIndex) >= len(btcTx.Vin) {
				log.Printf("WARNING: Spend index %d out of range for transaction %s (has %d inputs)", spendIndex, tx.TxID.String(), len(btcTx.Vin))
				continue
			}

			vin := btcTx.Vin[spendIndex]
			if vin.HashPrevout == nil || vin.Coinbase != nil {
				continue // Skip coinbase inputs
			}

			// Find the space pointer being spent
			spacePointer, err := q.FindSpacePointerByOutpoint(ctx, db.FindSpacePointerByOutpointParams{
				Txid: *vin.HashPrevout,
				Vout: int32(vin.IndexPrevout),
			})
			if err != nil {
				log.Printf("WARNING: Space pointer not found for prevout %s:%d - Error: %v", vin.HashPrevout.String(), vin.IndexPrevout, err)
				continue
			}

			// Update the spender fields
			err = q.UpdateSpacePointerSpender(ctx, db.UpdateSpacePointerSpenderParams{
				SpentBlockHash: &blockHash,
				SpentTxid:      &tx.TxID,
				SpentVin:       pgtype.Int4{Int32: int32(spendIndex), Valid: true},
				BlockHash:      spacePointer.BlockHash,
				Txid:           spacePointer.Txid,
				Vout:           spacePointer.Vout,
			})
			if err != nil {
				log.Printf("WARNING: Failed to update space pointer spender for %s:%d - Error: %v", vin.HashPrevout.String(), vin.IndexPrevout, err)
			}
		}
	}

	for _, commitment := range tx.Commitments {
		if commitment.Space[0] == '@' {
			commitment.Space = commitment.Space[1:]
		}
		err := q.InsertCommitment(ctx, db.InsertCommitmentParams{
			BlockHash:   blockHash,
			Txid:        tx.TxID,
			Name:        commitment.Space,
			StateRoot:   &commitment.StateRoot,
			HistoryHash: &commitment.HistoryHash,
			Revocation:  false,
		})
		if err != nil {
			return sqlTx, fmt.Errorf("failed to insert commitment: %w", err)
		}
	}

	for _, revokedCommitment := range tx.RevokedCommitments {
		if revokedCommitment.Space[0] == '@' {
			revokedCommitment.Space = revokedCommitment.Space[1:]
		}
		exists, err := q.CommitmentExists(ctx, db.CommitmentExistsParams{
			Name:      revokedCommitment.Space,
			StateRoot: &revokedCommitment.StateRoot,
		})
		if err != nil {
			return sqlTx, fmt.Errorf("failed to check revoked commitment existence: %w", err)
		}
		if !exists {
			log.Printf("WARNING: Revoked commitment not found, Space: %s, StateRoot: %s, TxID: %s", revokedCommitment.Space, revokedCommitment.StateRoot.String(), tx.TxID.String())
		}
		err = q.InsertCommitment(ctx, db.InsertCommitmentParams{
			BlockHash:   blockHash,
			Txid:        tx.TxID,
			Name:        revokedCommitment.Space,
			StateRoot:   &revokedCommitment.StateRoot,
			HistoryHash: &revokedCommitment.HistoryHash,
			Revocation:  true,
		})
		if err != nil {
			return sqlTx, fmt.Errorf("failed to insert revoked commitment: %w", err)
		}
	}

	for _, ptrOut := range tx.Creates {
		if ptrOut.ID != nil {
			var data *Bytes
			if ptrOut.Data != nil {
				data = ptrOut.Data
			}

			err := q.InsertSpacePointer(ctx, db.InsertSpacePointerParams{
				BlockHash:    blockHash,
				Txid:         tx.TxID,
				Vout:         int32(ptrOut.N),
				Sptr:         *ptrOut.ID,
				Value:        int64(ptrOut.Value),
				ScriptPubkey: ptrOut.ScriptPubkey,
				Data:         data,
			})
			if err != nil {
				return sqlTx, fmt.Errorf("failed to insert space pointer: %w", err)
			}
		}
	}

	for _, delegation := range tx.NewDelegations {
		name := delegation.Space
		if name[0] == '@' {
			name = name[1:]
		}

		// Find the vout index for this delegation by matching the SptrKey
		var vout int32
		for _, ptrOut := range tx.Creates {
			if ptrOut.ID != nil && *ptrOut.ID == delegation.Sptr {
				vout = int32(ptrOut.N)
				break
			}
		}

		err := q.InsertDelegation(ctx, db.InsertDelegationParams{
			Sptr:      delegation.Sptr,
			Name:      name,
			BlockHash: blockHash,
			Txid:      tx.TxID,
			Vout:      vout,
		})
		if err != nil {
			return sqlTx, fmt.Errorf("failed to insert delegation: %w", err)
		}
	}

	// Mark revoked delegations
	for _, delegation := range tx.RevokedDelegations {
		name := delegation.Space
		if name[0] == '@' {
			name = name[1:]
		}

		// Find the latest active delegation for this sptr and name
		existingDelegation, err := q.FindLatestDelegationBySptr(ctx, db.FindLatestDelegationBySptrParams{
			Sptr: delegation.Sptr,
			Name: name,
		})
		if err != nil {
			log.Printf("WARNING: Could not find delegation to revoke - Space: %s, Sptr: %s, Error: %v", name, delegation.Sptr, err)
			continue
		}

		err = q.UpdateDelegationRevoked(ctx, db.UpdateDelegationRevokedParams{
			RevokedBlockHash: &blockHash,
			RevokedTxid:      &tx.TxID,
			RevokedVout:      pgtype.Int4{Int32: 0, Valid: true}, //now we don't have output index set
			BlockHash:        existingDelegation.BlockHash,
			Txid:             existingDelegation.Txid,
			Vout:             existingDelegation.Vout,
		})
		if err != nil {
			return sqlTx, fmt.Errorf("failed to revoke delegation: %w", err)
		}
	}

	return sqlTx, nil
}

func StoreSpacesTransaction(ctx context.Context, tx node.MetaTransaction, blockHash Bytes, sqlTx pgx.Tx) (pgx.Tx, error) {
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

		if err := q.InsertVMetaOut(ctx, vmet); err != nil {
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

		if err := q.InsertVMetaOut(ctx, vmet); err != nil {
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

func StoreBitcoinBlock(ctx context.Context, block *node.Block, tx pgx.Tx) (pgx.Tx, error) {
	q := db.New(tx)
	blockParams := db.UpsertBlockParams{}
	copier.Copy(&blockParams, &block)
	wasInserted, err := q.UpsertBlock(ctx, blockParams)
	if err != nil {
		return tx, err
	}
	if wasInserted {
		// Prepare all transactions for batch insert
		batchParams := prepareBatchTransactions(block.Transactions, &blockParams.Hash)

		log.Printf("Batch inserting %d transactions for block %s", len(batchParams), &blockParams.Hash)

		// Batch insert all transactions at once using PostgreSQL COPY protocol
		rowsAffected, err := q.InsertBatchTransactions(ctx, batchParams)
		if err != nil {
			return tx, fmt.Errorf("batch insert transactions: %w", err)
		}

		log.Printf("Successfully inserted %d transactions", rowsAffected)
	}
	return tx, nil
}

func storeTransactionBase(ctx context.Context, q *db.Queries, transaction *node.Transaction, blockHash *Bytes, txIndex *int32) error {
	inputCount, outputCount, totalOutputValue := calculateAggregates(transaction)
	if blockHash.String() != deadbeefString {
		params := db.InsertTransactionParams{}
		copier.Copy(&params, transaction)
		params.BlockHash = *blockHash
		params.Index = *txIndex
		params.InputCount = inputCount
		params.OutputCount = outputCount
		params.TotalOutputValue = totalOutputValue
		return q.InsertTransaction(ctx, params)
	}
	params := db.InsertMempoolTransactionParams{}
	copier.Copy(&params, transaction)
	params.BlockHash = *blockHash
	params.InputCount = inputCount
	params.OutputCount = outputCount
	params.TotalOutputValue = totalOutputValue
	return q.InsertMempoolTransaction(ctx, params)
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
func GetSyncedHead(ctx context.Context, pg *pgx.Conn, bc *node.BitcoinClient) (int32, *Bytes, error) {
	q := db.New(pg)
	//takes last block from the DB
	height, err := q.GetBlocksMaxHeight(ctx)
	if err != nil {
		return -1, nil, err
	}
	//height is the height of the db block
	for height >= 0 {
		//take last block hash from the DB
		dbHash, err := q.GetBlockHashByHeight(ctx, height)
		if err != nil {
			return -1, nil, err
		}
		//takes the block of same height from the bitcoin node
		// modify that if it doesn't work we descend by
		nodeHash, err := bc.GetBlockHash(ctx, int(height))
		if err != nil {
			//do we need that?
			if strings.Contains(err.Error(), "Block height out of range") {
				height -= 1
				continue
			}
			return -1, nil, err
		}
		// nodeHash *bytes
		// dbHash Bytes
		if bytes.Equal(dbHash, *nodeHash) {
			//marking all the blocks in the DB after the sycned height as orphans
			if err := q.SetOrphanAfterHeight(ctx, height); err != nil {
				return -1, nil, err
			}
			if err := q.SetNegativeHeightToOrphans(ctx); err != nil {
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
	tx, err = StoreBitcoinBlock(ctx, block, tx)
	if err != nil {
		return err
	}

	if block.Height >= activationBlock {
		spacesBlock, err := sc.GetBlockMeta(ctx, block.Hash.String())
		if err != nil {
			return err
		}

		tx, err = StoreSpacesTransactions(ctx, spacesBlock.Transactions, block.Hash, tx)
		if err != nil {
			return err
		}

		spacesPtrBlock, err := sc.GetPtrBlockMeta(ctx, block.Hash.String())
		if err != nil {
			return err
		}

		tx, err = StoreSpacesPtrTransactions(ctx, spacesPtrBlock.Transactions, block.Transactions, block.Hash, tx)
		if err != nil {
			return err
		}

	}

	return tx.Commit(ctx)
}

func StoreTransaction(ctx context.Context, q *db.Queries, transaction *node.Transaction, blockHash *Bytes, txIndex *int32) error {
	if err := storeTransactionBase(ctx, q, transaction, blockHash, txIndex); err != nil {
		return err
	}
	return nil
}
