package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	_ "github.com/lib/pq"
	"github.com/spacesprotocol/explorer-indexer/pkg/db"
	"github.com/spacesprotocol/explorer-indexer/pkg/node"
	"github.com/spacesprotocol/explorer-indexer/pkg/store"
)

var activationBlock = getActivationBlock()
var syncEndHeight = getSyncEndHeight()

const blockCountToCommit = 5000

func getActivationBlock() int32 {
	if height := os.Getenv("ACTIVATION_BLOCK_HEIGHT"); height != "" {
		if h, err := strconv.ParseInt(height, 10, 32); err == nil {
			return int32(h)
		}
	}
	return 0
}

func getSyncEndHeight() int32 {
	if height := os.Getenv("SYNC_END_HEIGHT"); height != "" {
		if h, err := strconv.ParseInt(height, 10, 32); err == nil {
			return int32(h)
		}
	}
	return -1 // -1 means sync to the latest block
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// Check if activation block is set
	if activationBlock <= 0 {
		log.Fatalln("ACTIVATION_BLOCK_HEIGHT environment variable must be set and greater than 0")
	}

	// Initialize clients
	bitcoinClient := node.NewClient(os.Getenv("BITCOIN_NODE_URI"), os.Getenv("BITCOIN_NODE_USER"), os.Getenv("BITCOIN_NODE_PASSWORD"))
	spacesClient := node.NewClient(os.Getenv("SPACES_NODE_URI"), os.Getenv("RPC_USER"), os.Getenv("RPC_PASSWORD"))
	bc := node.BitcoinClient{Client: bitcoinClient}
	sc := node.SpacesClient{Client: spacesClient}

	pg, err := pgx.Connect(context.Background(), os.Getenv("POSTGRES_URI"))
	if err != nil {
		log.Fatalln(err)
	}
	defer pg.Close(context.Background())

	// Get retry settings
	updateInterval, err := strconv.Atoi(os.Getenv("UPDATE_DB_INTERVAL"))
	if err != nil {
		updateInterval = 5 // default to 5 seconds
	}

	for {
		if err := syncSpacesTransactions(pg, &bc, &sc); err != nil {
			log.Printf("Sync failed: %v. Retrying in %d seconds...", err, updateInterval)
			time.Sleep(time.Duration(updateInterval) * time.Second)
			continue
		}
		log.Print("Spaces transactions sync completed successfully")
		return
	}
}

func syncSpacesTransactions(pg *pgx.Conn, bc *node.BitcoinClient, sc *node.SpacesClient) error {
	ctx := context.Background()

	// Create a transaction for tracking sync progress
	tx, err := pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	q := db.New(tx)
	// Get current chain height if end height is not specified
	endHeight := syncEndHeight
	if endHeight == -1 {
		blockCount, err := q.GetBlocksMaxHeight(ctx)
		if err != nil {
			return err
		}
		endHeight = int32(blockCount)
		log.Printf("Using current chain height: %d", endHeight)
	}

	log.Printf("Starting spaces transactions sync from block %d to %d", activationBlock, endHeight)

	for height := activationBlock; height <= endHeight; height++ {
		start := time.Now()
		blockHash, err := bc.GetBlockHash(ctx, int(height))
		if err != nil {
			return err
		}

		spacesBlock, err := sc.GetBlockMeta(ctx, blockHash.String())
		if err != nil {
			return err
		}

		txCount := len(spacesBlock.Transactions)

		for txIndex, spaceTx := range spacesBlock.Transactions {
			if txIndex > 0 && txIndex%50 == 0 {
				log.Printf("  Processed %d/%d spaces transactions in block %d", txIndex, txCount, height)
			}

			tx, err = store.StoreSpacesTransaction(ctx, spaceTx, *blockHash, tx)
			if err != nil {
				return err
			}
		}

		spacesPtrBlock, err := sc.GetPtrBlockMeta(ctx, blockHash.String())
		if err != nil {
			return err
		}

		ptrTxCount := len(spacesPtrBlock.Transactions)

		for _, ptrTx := range spacesPtrBlock.Transactions {

			btcTx, err := bc.GetTransaction(ctx, ptrTx.TxID.String())
			if err != nil {
				return err
			}

			tx, err = store.StoreSpacesPtrTransaction(ctx, ptrTx, btcTx, *blockHash, tx)
			if err != nil {
				return err
			}
		}

		// Commit every N blocks to avoid large transactions
		if height%blockCountToCommit == 0 {
			elapsed := time.Since(start)
			log.Printf("Block %d completed in %s with %d spaces tx and %d ptr tx", height, elapsed, txCount, ptrTxCount)
			if err := tx.Commit(ctx); err != nil {
				return err
			}

			tx, err = pg.Begin(ctx)
			if err != nil {
				return err
			}
			defer tx.Rollback(ctx)

			log.Printf("Committed progress at block %d", height)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	log.Printf("Successfully synced spaces transactions from block %d to %d", activationBlock, endHeight)
	return nil
}
