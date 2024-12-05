package main

import (
	"bytes"
	"context"
	"database/sql"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/spacesprotocol/explorer-backend/pkg/db"
	"github.com/spacesprotocol/explorer-backend/pkg/node"
	mysync "github.com/spacesprotocol/explorer-backend/pkg/sync"

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

// TODO overwrite already synced blocks after the initial fast sync start
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	bitcoinClient := node.NewClient(os.Getenv("BITCOIN_NODE_URI"), os.Getenv("BITCOIN_NODE_USER"), os.Getenv("BITCOIN_NODE_PASSWORD"))

	bc := node.BitcoinClient{Client: bitcoinClient}

	pg, err := sql.Open("postgres", os.Getenv("POSTGRES_URI"))
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

func syncGapBlocks(pg *sql.DB, bc *node.BitcoinClient) error {
	var hash *Bytes
	var gapEnd int32

	var gapStarted bool
	height, hash, err := getSyncedHead(pg, bc)
	if err != nil {
		return err
	}
	log.Printf("found synced block of height %d and hash %s", height, hash)

	for workingHeight := height; workingHeight >= 0; workingHeight-- {
		q := db.New(pg)
		ctx := context.Background()
		block, err := q.GetBlockByHeight(ctx, workingHeight)
		if err == nil {
			if !gapStarted {
				log.Printf("goin down, got block at height %d, it has hash %s", workingHeight, block.Hash)
				continue
			}
			gapEnd = workingHeight
			break
		}
		if err == sql.ErrNoRows {
			gapStarted = true
			log.Printf("found no rows at height %d", workingHeight)
		}
	}
	log.Printf("gap end is on the height of %d", gapEnd)
	height = gapEnd

	hash, err = bc.GetBlockHash(context.Background(), int(height))
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
		sqlTx, err := pg.BeginTx(context.Background(), nil)
		if err != nil {
			return err
		}

		sqlTx, err = mysync.SyncBlock(block, sqlTx)
		if err != nil {
			sqlTx.Rollback()
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
		err = sqlTx.Commit()
		if err != nil {
			return err
		}
		nextBlockHash = block.NextBlockHash
	}
	return nil

}

// detects chain split (reorganization) and
// returns the height and blockhash of the last block that is identical in the db and in the node
func getSyncedHead(pg *sql.DB, bc *node.BitcoinClient) (int32, *Bytes, error) {
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
