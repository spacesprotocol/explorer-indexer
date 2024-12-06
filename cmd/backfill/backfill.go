package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/spacesprotocol/explorer-backend/pkg/db"
	"github.com/spacesprotocol/explorer-backend/pkg/node"
	"github.com/spacesprotocol/explorer-backend/pkg/store"

	_ "github.com/lib/pq"
	. "github.com/spacesprotocol/explorer-backend/pkg/types"
)

var activationBlock = getActivationBlock()
var fastSyncBlockHeight = getFastSyncBlockHeight()

func getActivationBlock() int32 {
	if height := os.Getenv("ACTIVATION_BLOCK_HEIGHT"); height != "" {
		if h, err := strconv.ParseInt(height, 10, 32); err == nil {
			return int32(h)
		}
	}
	return 0
}

func getFastSyncBlockHeight() int32 {
	if height := os.Getenv("FAST_SYNC_BLOCK_HEIGHT"); height != "" {
		if h, err := strconv.ParseInt(height, 10, 32); err == nil {
			return int32(h)
		}
	}
	return 0
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	bitcoinClient := node.NewClient(os.Getenv("BITCOIN_NODE_URI"), os.Getenv("BITCOIN_NODE_USER"), os.Getenv("BITCOIN_NODE_PASSWORD"))

	bc := node.BitcoinClient{Client: bitcoinClient}

	pg, err := pgx.Connect(context.Background(), os.Getenv("POSTGRES_URI"))
	if err != nil {
		log.Fatalln(err)
	}

	updateInterval, err := strconv.Atoi(os.Getenv("UPDATE_DB_INTERVAL"))
	if err != nil {
		log.Fatalln(err)
	}

	for {

		if err := syncGapBlocks(pg, &bc); err != nil {
			log.Println(err)
		}
		time.Sleep(time.Duration(updateInterval) * time.Second)
	}

}

func syncGapBlocks(pg *pgx.Conn, bc *node.BitcoinClient) error {
	ctx := context.Background()
	var hash *Bytes
	var gapEnd int32

	var gapStarted bool = false
	height, hash, err := store.GetSyncedHead(pg, bc)
	if err != nil {
		return err
	}
	log.Printf("found synced block of height %d and hash %s", height, hash)

	q := db.New(pg)
	for workingHeight := height; workingHeight >= 0; workingHeight-- {
		ctx := context.Background()
		block, err := q.GetBlockByHeight(ctx, workingHeight)

		if err == nil {
			if !gapStarted {
				log.Printf("going down, got block at height %d, it has hash %s", workingHeight, block.Hash)
				continue
			}
			gapEnd = workingHeight
			break
		}

		if err == pgx.ErrNoRows {
			gapStarted = true
			log.Printf("found no rows at the height %d", workingHeight)
		} else {
			log.Fatal(err)
		}
	}
	log.Printf("the gap end is at the height of %d", gapEnd)
	hash, err = bc.GetBlockHash(context.Background(), int(gapEnd))
	if err != nil {
		return err
	}

	hashString, err := hash.MarshalText()
	if err != nil {
		return err
	}

	block, err := bc.GetBlock(context.Background(), string(hashString))
	if err != nil {
		return err
	}
	nextBlockHash := block.NextBlockHash

	for nextBlockHash != nil {

		nextHashString, err := nextBlockHash.MarshalText()
		if err != nil {
			return err
		}
		block, err := bc.GetBlock(context.Background(), string(nextHashString))
		if err != nil {
			return err
		}
		log.Printf("trying to sync block #%d", block.Height)
		sqlTx, err := pg.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return err
		}
		defer func() {
			err := sqlTx.Rollback(ctx)
			if err != nil && err != pgx.ErrTxClosed {
				log.Fatalf("block sync: cannot rollback sql transaction: %s", err)
			}
		}()

		if block.Height >= fastSyncBlockHeight {
			sqlTx, err = store.UpdateBlockSpender(block, sqlTx)
		} else {
			sqlTx, err = store.StoreBlock(block, sqlTx)
		}
		if err != nil {
			return err
		}
		// if block.Height >= activationBlock {
		// 	spacesBlock, err := sc.GetBlockMeta(context.Background(), string(nextHashString))
		// 	if err != nil {
		// 		return err
		// 	}
		// 	sqlTx, err = syncSpacesTransactions(spacesBlock.Transactions, block.Hash, sqlTx)
		// 	if err != nil {
		// 		sqlTx.Rollback()
		// 		return err
		// 	}
		// }
		err = sqlTx.Commit(ctx)
		if err != nil {
			return err
		}
		nextBlockHash = block.NextBlockHash
	}
	return nil

}
