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
	spacesClient := node.NewClient(os.Getenv("SPACES_NODE_URI"), "test", "test")

	sc := node.SpacesClient{Client: spacesClient}
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

		if err := syncBlocks(pg, &bc, &sc); err != nil {
			log.Println(err)
			time.Sleep(time.Second)
		}
		time.Sleep(time.Duration(updateInterval) * time.Second)
	}

}

func syncRollouts(ctx context.Context, pg *pgx.Conn, sc *node.SpacesClient) error {
	tx, err := pg.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		err := tx.Rollback(ctx)
		if err != nil && err != pgx.ErrTxClosed {
			log.Fatalf("rollouts sync: cannot rollback sql transaction: %s", err)
		}
	}()

	q := db.New(tx)
	if err = q.DeleteRollouts(ctx); err != nil {
		return err
	}
	for i := 0; i < 10; i++ {
		result, err := sc.GetRollOut(ctx, i)
		if err != nil {
			return err
		}

		params := db.InsertRolloutParams{}
		for _, space := range *result {
			if space.Name[0] == '@' {
				params.Name = space.Name[1:]
			} else {
				log.Fatalf("found incorrect space name during rollout sync: %s", space.Name)
			}
			params.Bid = int64(space.Value)
			params.Target = int64(i)
			if err := q.InsertRollout(ctx, params); err != nil {
				log.Printf("error inserting rollout batch %d: %v", i, err)
				return err
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

func syncBlocks(pg *pgx.Conn, bc *node.BitcoinClient, sc *node.SpacesClient) error {
	ctx := context.Background()
	var hash *Bytes
	height, hash, err := store.GetSyncedHead(pg, bc)
	if err != nil {
		return err
	}
	log.Printf("found synced block of height %d and hash %s", height, hash)

	if height < fastSyncBlockHeight {
		hash, err = bc.GetBlockHash(ctx, int(fastSyncBlockHeight-1))
		if err != nil {
			return err
		}
	}

	hashString, err := hash.MarshalText()
	if err != nil {
		return err
	}

	block, err := bc.GetBlock(ctx, string(hashString))
	if err != nil {
		return err
	}
	nextBlockHash := block.NextBlockHash

	if block.Height >= activationBlock {
		if err := syncRollouts(ctx, pg, sc); err != nil {
			log.Println(err)
			return err
		}
	}

	for nextBlockHash != nil {
		nextHashString, err := nextBlockHash.MarshalText()
		if err != nil {
			return err
		}
		block, err := bc.GetBlock(ctx, string(nextHashString))
		if err != nil {
			return err
		}
		log.Printf("trying to sync block #%d", block.Height)

		tx, err := pg.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return err
		}
		defer func() {
			err := tx.Rollback(ctx)
			if err != nil && err != pgx.ErrTxClosed {
				log.Fatalf("block sync: cannot rollback sql transaction: %s", err)
			}
		}()

		tx, err = store.StoreBlock(block, tx)
		if err != nil {
			return err
		}
		if block.Height >= activationBlock {
			spacesBlock, err := sc.GetBlockMeta(ctx, string(nextHashString))
			if err != nil {
				return err
			}
			tx, err = store.StoreSpacesTransactions(spacesBlock.Transactions, block.Hash, tx)
			if err != nil {
				return err
			}
		}
		err = tx.Commit(ctx)
		if err != nil {
			return err
		}
		nextBlockHash = block.NextBlockHash
	}
	return nil
}
