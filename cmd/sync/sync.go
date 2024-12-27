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
	// Get current mempool state and existing mempool transactions
	currentTxIds, err := bc.GetMempoolTxIds(context.Background())
	if err != nil {
		return err
	}
	log.Printf("found %d txs in current mempool", len(currentTxIds))

	q := db.New(pg)
	existingTxidsBytes, err := q.GetMempoolTxids(context.Background())
	if err != nil {
		return err
	}

	existingTxs := make(map[string]struct{})
	for _, txid := range existingTxidsBytes {
		existingTxs[txid.String()] = struct{}{}
	}
	log.Printf("found %d txs in database mempool", len(existingTxs))

	// Find transactions to delete and add
	var toDelete []Bytes
	for _, txidBytes := range existingTxidsBytes {
		txidStr := txidBytes.String()
		found := false
		for _, currentTxid := range currentTxIds {
			if txidStr == currentTxid {
				found = true
				break
			}
		}
		if !found {
			toDelete = append(toDelete, txidBytes)
		}
	}

	var toAdd []string
	for _, txid := range currentTxIds {
		if _, exists := existingTxs[txid]; !exists {
			toAdd = append(toAdd, txid)
		}
	}

	// Delete old transactions in a single transaction
	if len(toDelete) > 0 {
		log.Printf("deleting %d old transactions", len(toDelete))
		sqlTx, err := pg.BeginTx(context.Background(), pgx.TxOptions{})
		if err != nil {
			return err
		}
		defer sqlTx.Rollback(context.Background())

		qtx := db.New(sqlTx)
		for _, txid := range toDelete {
			if err := qtx.DeleteMempoolTxInputsByTxid(context.Background(), txid); err != nil {
				return err
			}
			if err := qtx.DeleteMempoolTxOutputsByTxid(context.Background(), txid); err != nil {
				return err
			}
			if err := qtx.DeleteMempoolTransactionByTxid(context.Background(), txid); err != nil {
				return err
			}
		}
		if err := sqlTx.Commit(context.Background()); err != nil {
			return err
		}
	}

	if len(toAdd) == 0 {
		log.Printf("no new transactions to add")
		return nil
	}

	var deadbeef Bytes
	deadbeef.UnmarshalString(deadbeefString)

	// Process new transactions in chunks, with separate transaction per chunk
	log.Printf("processing %d new transactions", len(toAdd))

	for i := 0; i < len(toAdd); i += mempoolChunkSize {
		end := i + mempoolChunkSize
		if end > len(toAdd) {
			end = len(toAdd)
		}
		log.Printf("processing chunk #%d of new mempool txs", i/mempoolChunkSize+1)

		// Start new transaction for this chunk
		sqlTx, err := pg.BeginTx(context.Background(), pgx.TxOptions{})
		if err != nil {
			return err
		}
		defer sqlTx.Rollback(context.Background())

		q = db.New(sqlTx)
		var hexes []string
		chunk := toAdd[i:end]

		for _, txid := range chunk {
			transaction, err := bc.GetTransaction(context.Background(), txid)
			if err != nil {
				continue
			}
			hexes = append(hexes, transaction.Hex.String())

			if err := store.StoreTransaction(q, transaction, &deadbeef, nil); err != nil {
				return err
			}
		}

		// Process spaces transactions for this chunk
		if len(hexes) > 0 {
			metaTxs, err := sc.CheckPackage(context.Background(), hexes)
			if err != nil {
				return err
			}
			for _, metaTx := range metaTxs {
				if metaTx != nil {
					if sqlTx, err = store.StoreSpacesTransaction(*metaTx, deadbeef, sqlTx); err != nil {
						return err
					}
				}
			}
		}

		// Commit this chunk's transaction
		if err := sqlTx.Commit(context.Background()); err != nil {
			return err
		}
	}

	return nil
}
