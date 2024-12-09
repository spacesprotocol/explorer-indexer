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
	spacesClient := node.NewClient(os.Getenv("SPACES_NODE_URI"), "test", "test")

	bc := node.BitcoinClient{Client: bitcoinClient}
	sc := node.SpacesClient{Client: spacesClient}

	pg, err := pgx.Connect(context.Background(), os.Getenv("POSTGRES_URI"))
	if err != nil {
		log.Fatalln(err)
	}

	updateInterval, err := strconv.Atoi(os.Getenv("UPDATE_DB_INTERVAL"))
	if err != nil {
		log.Fatalln(err)
	}

	for {

		if err := syncGapBlocks(pg, &bc, &sc); err != nil {
			log.Println(err)
			time.Sleep(time.Duration(updateInterval) * time.Second)
			continue
		}
		log.Print("gap has been filled")
		return
	}
}

func syncGapBlocks(pg *pgx.Conn, bc *node.BitcoinClient, sc *node.SpacesClient) error {
	ctx := context.Background()
	var gapStarted bool = false
	var height, gapBeginning int32

	maxHeight, hash, err := store.GetSyncedHead(pg, bc)
	if err != nil {
		return err
	}
	log.Printf("found synced block of height %d and hash %s", maxHeight, hash)

	q := db.New(pg)
	for height = maxHeight; ; height-- {
		if height <= -1 {
			height = -1
			break
		}

		block, err := q.GetBlockByHeight(ctx, height)
		if err == nil {
			if !gapStarted {
				log.Printf("going down, got block at height %d with hash %s", height, block.Hash)
				continue
			}
			break
		}
		if err == pgx.ErrNoRows {
			if !gapStarted {
				gapBeginning = height
				log.Print("gap began at ", gapBeginning)
			}
			gapStarted = true
			log.Printf("found gap at height %d", height)
			continue
		}
		return err
	}

	if !gapStarted {
		log.Printf("no gaps found")
		return nil
	}

	log.Printf("the gap end is at the height of %d", height)
	nextBlockHash, err := bc.GetBlockHash(context.Background(), int(height+1))
	if err != nil {
		return err
	}

	for nextBlockHash != nil {
		block, err := bc.GetBlock(ctx, nextBlockHash.String())
		if err != nil {
			return err
		}

		if block.Height <= gapBeginning {
			if err := store.StoreBlock(ctx, pg, block, sc, activationBlock); err != nil {
				return err
			}
		}

		sqlTx, err := pg.Begin(ctx)
		defer sqlTx.Rollback(ctx)
		if err != nil {
			return err
		}

		if sqlTx, err = store.UpdateBlockSpender(block, sqlTx); err != nil {
			return err
		}
		sqlTx.Commit(ctx)

		nextBlockHash = &block.NextBlockHash

	}
	return nil

}
