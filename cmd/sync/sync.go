package main

import (
	"context"
	"strings"

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
var mempoolChunkSize = getMempoolChunkSize()

const deadbeefString = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
const mempoolSyncTimeout = 60 //in seconds

func getMempoolChunkSize() int {
	if height := os.Getenv("MEMPOOL_CHUNK_SIZE"); height != "" {
		if h, err := strconv.ParseInt(height, 10, 32); err == nil {
			return int(h)
		}
	}
	return 200
}

func getActivationBlock() int32 {
	if height := os.Getenv("ACTIVATION_BLOCK_HEIGHT"); height != "" {
		if h, err := strconv.ParseInt(height, 10, 32); err == nil {
			return int32(h)
		}
	}
	return -1
}

func getFastSyncBlockHeight() int32 {
	if height := os.Getenv("FAST_SYNC_BLOCK_HEIGHT"); height != "" {
		if h, err := strconv.ParseInt(height, 10, 32); err == nil {
			return int32(h) - 1
		}
	}
	return -1
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	bitcoinClient := node.NewClient(os.Getenv("BITCOIN_NODE_URI"), os.Getenv("BITCOIN_NODE_USER"), os.Getenv("BITCOIN_NODE_PASSWORD"))
	spacesClient := node.NewClient(os.Getenv("SPACES_NODE_URI"), "test", "test")

	sc := node.SpacesClient{Client: spacesClient}
	bc := node.BitcoinClient{Client: bitcoinClient}

	updateInterval, err := strconv.Atoi(os.Getenv("UPDATE_DB_INTERVAL"))
	if err != nil {
		log.Fatalln(err)
	}

	for {
		connCtx, cancel := context.WithTimeout(context.Background(), time.Minute)

		pg, err := pgx.Connect(connCtx, os.Getenv("POSTGRES_URI"))
		cancel()

		if err != nil {
			log.Printf("failed to connect to database: %v", err)
			time.Sleep(time.Second)
			continue
		}

		defer pg.Close(context.Background())

		if err := syncBlocks(pg, &bc, &sc); err != nil {
			log.Println(err)
			pg.Close(context.Background())
			time.Sleep(time.Second)
			continue
		}

		if err := syncMempool(pg, &bc, &sc); err != nil {
			log.Println(err)
		}

		if err := pg.Close(context.Background()); err != nil {
			log.Printf("error closing connection: %v", err)
		}

		time.Sleep(time.Duration(updateInterval) * time.Second)
	}
}

func syncRollouts(ctx context.Context, pg *pgx.Conn, sc *node.SpacesClient) error {
	sqlTx, err := pg.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	defer sqlTx.Rollback(ctx)

	q := db.New(sqlTx)
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
	if err = sqlTx.Commit(ctx); err != nil {
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
		height = fastSyncBlockHeight
	}

	height++
	log.Print("trying to get block ", height)
	hash, err = bc.GetBlockHash(ctx, int(height))
	if err != nil {
		if strings.Contains(err.Error(), "Block height out of range") {
			return nil
		}
		return err
	}

	block, err := bc.GetBlock(ctx, hash.String())
	if err != nil {
		return err
	}

	if block.Height >= activationBlock {
		if err := syncRollouts(ctx, pg, sc); err != nil {
			log.Println(err)
			return err
		}
	}

	if err := store.StoreBlock(ctx, pg, block, sc, activationBlock); err != nil {
		return err
	}
	nextBlockHash := block.NextBlockHash

	for nextBlockHash != nil {
		block, err := bc.GetBlock(ctx, nextBlockHash.String())
		if err != nil {
			return err
		}

		if err := store.StoreBlock(ctx, pg, block, sc, activationBlock); err != nil {
			return err
		}

		nextBlockHash = block.NextBlockHash
	}
	return nil
}
