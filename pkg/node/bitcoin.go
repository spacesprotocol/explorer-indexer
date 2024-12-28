package node

import (
	"context"
	"log"
	"sort"

	. "github.com/spacesprotocol/explorer-backend/pkg/types"
)

var mempoolChunkSize = 200

type BitcoinClient struct {
	*Client
}

func (client *BitcoinClient) GetBlockChainInfo() {
	ctx := context.Background()
	var x interface{}
	// var z []string
	err := client.Rpc(ctx, "getblockchaininfo", []interface{}{}, x)
	log.Print(err)

}

func (client *BitcoinClient) GetBlock(ctx context.Context, blockHash string) (*Block, error) {
	block := new(Block)
	err := client.Rpc(ctx, "getblock", []interface{}{blockHash, 2}, block)
	if err != nil {
		return nil, err
	}
	return block, err
}

func (client *BitcoinClient) GetBlockHash(ctx context.Context, height int) (*Bytes, error) {
	blockHash := new(Bytes)
	// hexHeight := fmt.Sprintf("%x", height)
	err := client.Rpc(ctx, "getblockhash", []interface{}{height}, blockHash)
	if err != nil {
		return nil, err
	}
	return blockHash, err
}

func (client *BitcoinClient) GetBestBlockHeight(ctx context.Context) (int32, Bytes, error) {
	blockHash, err := client.GetBestBlockHash(ctx)
	if err != nil {
		return -1, nil, err
	}
	blockH, err := blockHash.MarshalText()
	if err != nil {
		return -1, nil, err
	}
	block, err := client.GetBlock(ctx, string(blockH))
	if err != nil {
		return -1, nil, err
	}
	return block.Height, block.Hash, nil
}

func (client *BitcoinClient) GetBestBlockHash(ctx context.Context) (*Bytes, error) {
	blockHash := new(Bytes)
	// hexHeight := fmt.Sprintf("%x", height)
	err := client.Rpc(ctx, "getbestblockhash", []interface{}{}, blockHash)
	if err != nil {
		return nil, err
	}
	return blockHash, err
}

func (client *BitcoinClient) GetTransaction(ctx context.Context, txId string) (*Transaction, error) {
	tx := new(Transaction)
	err := client.Rpc(ctx, "getrawtransaction", []interface{}{txId, 2}, tx)
	if err != nil {
		return nil, err
	}
	return tx, err
}

func (client *BitcoinClient) GetMempoolTxs(ctx context.Context) ([]Transaction, error) {
	var txids []string
	var txs []Transaction
	err := client.Rpc(ctx, "getrawmempool", nil, &txids)
	if err != nil {
		return nil, err
	}
	for _, txid := range txids {
		tx, err := client.GetTransaction(context.Background(), txid)
		if err != nil {
			return nil, err
		}
		txs = append(txs, *tx)
	}
	return txs, nil
}

type MempoolTx struct {
	Time    int64    `json:"time"`
	Depends []string `json:"depends"`
}

func (client *BitcoinClient) GetMempoolTxIds(ctx context.Context) ([][]string, error) {
	response := make(map[string]MempoolTx)
	err := client.Rpc(ctx, "getrawmempool", []interface{}{true}, &response)
	if err != nil {
		return nil, err
	}

	// Build dependency graph
	deps := make(map[string][]string) // txid -> list of txs that depend on it
	for txid, info := range response {
		for _, dep := range info.Depends {
			deps[dep] = append(deps[dep], txid)
		}
	}

	var orderedGroups [][]string
	processed := make(map[string]bool)

	// Process transactions with their dependencies into ordered groups
	var processGroup func(txid string) []string
	processGroup = func(txid string) []string {
		if processed[txid] {
			return nil
		}

		// Process this tx
		processed[txid] = true
		result := []string{txid}

		// Get all direct dependents and sort them by time
		dependents := deps[txid]
		sort.Slice(dependents, func(i, j int) bool {
			return response[dependents[i]].Time < response[dependents[j]].Time
		})

		// Process each dependent
		for _, dep := range dependents {
			if childGroup := processGroup(dep); childGroup != nil {
				result = append(result, childGroup...)
			}
		}
		return result
	}

	// Start with independent transactions
	var independentTxs []string
	for txid, info := range response {
		if !processed[txid] && len(info.Depends) == 0 {
			independentTxs = append(independentTxs, txid)
		}
	}

	// Sort independent transactions by time
	sort.Slice(independentTxs, func(i, j int) bool {
		return response[independentTxs[i]].Time < response[independentTxs[j]].Time
	})

	// Process each independent transaction and its dependency chain
	for _, txid := range independentTxs {
		if group := processGroup(txid); group != nil {
			orderedGroups = append(orderedGroups, group)
		}
	}

	// Handle any remaining transactions (cycles)
	for txid := range response {
		if !processed[txid] {
			if group := processGroup(txid); group != nil {
				orderedGroups = append(orderedGroups, group)
			}
		}
	}

	return orderedGroups, nil
}
