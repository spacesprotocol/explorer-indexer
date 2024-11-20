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
	spacesClient := node.NewClient(os.Getenv("SPACES_NODE_URI"), "test", "test")

	sc := node.SpacesClient{Client: spacesClient}
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

		if err := syncRollouts(pg, &sc); err != nil {
			log.Println(err)
			time.Sleep(time.Second)
			continue
		}

		if err := syncBlocks(pg, &bc, &sc); err != nil {
			log.Println(err)
			time.Sleep(time.Second)
			continue
		}
		time.Sleep(time.Duration(updateInterval) * time.Second)
	}

}

func syncRollouts(pg *sql.DB, sc *node.SpacesClient) error {
	ctx := context.Background()
	sqlTx, err := pg.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	q := db.New(sqlTx)
	q.DeleteRollouts(ctx)
	for i := 0; i < 10; i++ {
		result, err := sc.GetRollOut(ctx, i)
		if err != nil {
			log.Printf("error getting rollout for number %d: %v", i, err)
			sqlTx.Rollback()
			return err
		}

		params := db.InsertRolloutParams{}
		for _, space := range *result {
			if space.Name[0] == '@' {
				params.Name = space.Name[1:]
			} else {
				sqlTx.Rollback()
				log.Fatalf("found incorrect space name during rollout sync: %s", space.Name)
			}
			params.Bid = int64(space.Value)
			params.Target = int64(i)
			if err := q.InsertRollout(ctx, params); err != nil {
				log.Printf("Error inserting rollout batch %d: %v", i, err)
				sqlTx.Rollback()
				return err
			}
		}
	}
	if err = sqlTx.Commit(); err != nil {
		return err
	}
	return nil
}

func syncBlocks(pg *sql.DB, bc *node.BitcoinClient, sc *node.SpacesClient) error {
	var hash *Bytes
	height, hash, err := getSyncedHead(pg, bc)
	if err != nil {
		return err
	}
	log.Printf("found synced block of height %d and hash %s", height, hash)

	// if height == -1 {
	if height < fastSyncBlockHeight {
		hash, err = bc.GetBlockHash(context.Background(), int(fastSyncBlockHeight))
		if err != nil {
			return err
		}
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

	//TODO what if the node best block changes during the sync?
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

		sqlTx, err = syncBlock(block, sqlTx)
		if err != nil {
			sqlTx.Rollback()
			return err
		}
		if block.Height >= activationBlock {
			spacesBlock, err := sc.GetBlockMeta(context.Background(), string(nextHashString))
			if err != nil {
				return err
			}
			sqlTx, err = syncSpacesTransactions(spacesBlock.Transactions, block.Hash, sqlTx)
			if err != nil {
				sqlTx.Rollback()
				return err
			}
		}
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
