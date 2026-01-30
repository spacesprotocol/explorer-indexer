package main

import (
	"context"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/spacesprotocol/explorer-indexer/pkg/db"
	"github.com/spacesprotocol/explorer-indexer/pkg/node"
	"github.com/spacesprotocol/explorer-indexer/pkg/store"
	. "github.com/spacesprotocol/explorer-indexer/pkg/types"
)

func syncMempool(ctx context.Context, pg *pgx.Conn, bc *node.BitcoinClient, sc *node.SpacesClient) error {
	mempoolCtx, cancel := context.WithTimeout(ctx, mempoolSyncTimeout)
	defer cancel()

	ctx = mempoolCtx
	currentGroups, err := bc.GetMempoolTxIds(ctx)
	if err != nil {
		return err
	}

	// Build current mempool map
	nodeMempoolTxs := make(map[string]struct{})
	for _, group := range currentGroups {
		for _, txid := range group {
			nodeMempoolTxs[txid] = struct{}{}
		}
	}

	q := db.New(pg)
	existingTxidsBytes, err := q.GetMempoolTxids(ctx)
	log.Printf("found %d txs in db's mempool", len(existingTxidsBytes))
	if err != nil {
		return err
	}

	existingTxMap := make(map[string]Bytes, len(existingTxidsBytes))
	for _, txid := range existingTxidsBytes {
		existingTxMap[txid.String()] = txid
	}

	if err := cleanupMempoolTxs(ctx, pg, nodeMempoolTxs, existingTxMap); err != nil {
		return err
	}

	// Pre-filter groups that need processing
	var groupsToProcess [][]string
	for _, group := range currentGroups {
		if len(group) == 0 {
			continue
		}
		// Check if the dependent transaction (last in group) exists
		lastTxID := group[len(group)-1]
		if _, exists := existingTxMap[lastTxID]; !exists {
			groupsToProcess = append(groupsToProcess, group)
		}
	}

	log.Printf("filtered %d groups to process out of %d total groups", len(groupsToProcess), len(currentGroups))

	var deadbeef Bytes
	deadbeef.UnmarshalString(deadbeefString)

	// Process only the filtered groups
	for groupIndex, txGroup := range groupsToProcess {
		if groupIndex%50 == 0 {
			log.Printf("processing group #%d of %d", groupIndex, len(groupsToProcess))
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		sqlTx, err := pg.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return err
		}

		if err := processTxGroup(ctx, sqlTx, bc, sc, txGroup, deadbeef); err != nil {
			sqlTx.Rollback(ctx)
			return err
		}

		if err := sqlTx.Commit(ctx); err != nil {
			sqlTx.Rollback(ctx)
			return err
		}
	}

	return nil

}

func cleanupMempoolTxs(ctx context.Context, pg *pgx.Conn, nodeMempoolTxs map[string]struct{}, existingTxMap map[string]Bytes) error {
	var toDelete []Bytes
	for txidStr, txidBytes := range existingTxMap {
		if _, exists := nodeMempoolTxs[txidStr]; !exists {
			toDelete = append(toDelete, txidBytes)
		}
	}

	log.Printf("deleting %d txs from db's mempool", len(toDelete))
	if len(toDelete) > 0 {
		sqlTx, err := pg.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return err
		}
		defer sqlTx.Rollback(ctx)

		q := db.New(sqlTx)

		// Delete in chunks to avoid overwhelming the database
		chunkSize := mempoolChunkSize
		for i := 0; i < len(toDelete); i += chunkSize {
			end := i + chunkSize
			if end > len(toDelete) {
				end = len(toDelete)
			}
			chunk := toDelete[i:end]

			log.Printf("deleting chunk %d-%d of %d mempool txs", i+1, end, len(toDelete))
			if err := q.DeleteMempoolTransactionsByTxids(ctx, chunk); err != nil {
				return err
			}
		}

		log.Printf("successfully deleted %d mempool txs", len(toDelete))
		return sqlTx.Commit(ctx)
	}
	return nil
}

func processTxGroup(ctx context.Context, sqlTx pgx.Tx, bc *node.BitcoinClient, sc *node.SpacesClient, txGroup []string, deadbeef Bytes) error {
	q := db.New(sqlTx)
	var hexes []string

	for i, txid := range txGroup {
		tx, err := bc.GetTransaction(ctx, txid)
		if err != nil {
			log.Print("got error in the bitcoin node ", err)
			continue
		}
		hexes = append(hexes, tx.Hex.String())

		// Store only the last transaction (dependent one)
		if i == len(txGroup)-1 {
			if err := store.StoreTransaction(ctx, q, tx, &deadbeef, nil); err != nil {
				return err
			}
		}
	}

	if len(hexes) > 0 {
		metaTxs, err := sc.CheckPackage(ctx, hexes)
		if err != nil {
			return err
		}

		// Process the last metaTx since it's the dependent one
		if len(metaTxs) > 0 && metaTxs[len(metaTxs)-1] != nil {
			lastMetaTx := metaTxs[len(metaTxs)-1]
			if sqlTx, err = store.StoreSpacesTransaction(ctx, *lastMetaTx, deadbeef, sqlTx); err != nil {
				return err
			}

			// if sqlTx, err = store.StoreSpacesPtrTransaction(ctx, *lastMetaTx, deadbeef, sqlTx); err != nil {
			// 	return err
			// }
		}
	}
	return nil
}
