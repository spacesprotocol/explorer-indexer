package main

import (
	"context"
	"strings"

	"log"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jinzhu/copier"

	"github.com/spacesprotocol/explorer-backend/pkg/db"
	"github.com/spacesprotocol/explorer-backend/pkg/node"
	"github.com/spacesprotocol/explorer-backend/pkg/store"

	_ "github.com/lib/pq"
	. "github.com/spacesprotocol/explorer-backend/pkg/types"
)

var activationBlock = getActivationBlock()
var fastSyncBlockHeight = getFastSyncBlockHeight()

const deadbeefString = "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"

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

		if err := syncMempool(pg, &bc, &sc); err != nil {
			log.Println(err)
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

func syncMempool(pg *pgx.Conn, bc *node.BitcoinClient, sc *node.SpacesClient) error {

	txIds, err := bc.GetMempoolTxIds(context.Background())
	if err != nil {
		return err
	}
	sqlTx, err := pg.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer sqlTx.Rollback(context.Background())
	q := db.New(sqlTx)

	if err = q.DeleteMempoolVmetaouts(context.Background()); err != nil {
		return err
	}

	if err = q.DeleteMempoolTxOutputs(context.Background()); err != nil {
		return err
	}

	if err = q.DeleteMempoolTxInputs(context.Background()); err != nil {
		return err
	}

	if err = q.DeleteMempoolTransactions(context.Background()); err != nil {
		return err
	}

	tx := new(node.Transaction)
	metaTx := new(node.MetaTransaction)
	var deadbeef Bytes
	deadbeef.UnmarshalString(deadbeefString)

	for tx_index, txid := range txIds {
		ind := int32(tx_index)
		log.Printf("storing mempool tx %s and index %d", txid, ind)
		transaction, err := bc.GetTransaction(context.Background(), txid)
		if err != nil {
			return err
		}
		metaTransaction, err := sc.GetTxMeta(context.Background(), txid)
		// if err != nil {
		// 	return err
		// }

		copier.Copy(&tx, &transaction)
		err = store.StoreTransaction(q, tx, &deadbeef, &ind)
		if err != nil {
			return err
		}

		copier.Copy(&metaTx, metaTransaction)
		sqlTx, err = store.StoreSpacesTransaction(*metaTx, deadbeef, sqlTx)
		if err != nil {
			return err
		}
	}

	// for tx_index, transaction := range txs {
	// 	ind := int32(tx_index)
	// 	if err := store.StoreTransaction(q, &tx, &deadbeef, &ind); err != nil {
	// 		return err
	// 	}
	// }

	if err = sqlTx.Commit(context.Background()); err != nil {
		return err
	}
	return err
}
