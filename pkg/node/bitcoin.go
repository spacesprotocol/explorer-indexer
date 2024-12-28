package node

import (
	"context"
	"log"

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

	// Pre-allocate maps with expected size
	size := len(response)
	processed := make(map[string]struct{}, size)
	dependedBy := make(map[string][]string, size)

	// Single pass to build dependency index
	// Only build dependedBy as it's the critical path we need
	for txid, info := range response {
		for _, dep := range info.Depends {
			dependedBy[dep] = append(dependedBy[dep], txid)
		}
	}

	// Pre-allocate result slice
	var orderedGroups [][]string
	orderedGroups = make([][]string, 0, size/20) // Estimate avg group size of 20

	// Find independent transactions in a single pass with pre-allocated slice
	independentTxs := make([]string, 0, size/4) // Estimate 25% are independent
	for txid, info := range response {
		if len(info.Depends) == 0 {
			independentTxs = append(independentTxs, txid)
		}
	}

	// Use insertion sort for small batches instead of full sort
	// This is faster for small groups of transactions
	insertionSortByTime := func(txids []string) {
		for i := 1; i < len(txids); i++ {
			key := txids[i]
			keyTime := response[key].Time
			j := i - 1
			for j >= 0 && response[txids[j]].Time > keyTime {
				txids[j+1] = txids[j]
				j--
			}
			txids[j+1] = key
		}
	}

	// Process chains with minimal allocations
	chain := make([]string, 0, 100) // Pre-allocate typical chain size
	var processChain func(txid string)
	processChain = func(txid string) {
		if _, ok := processed[txid]; ok {
			return
		}

		processed[txid] = struct{}{}
		chain = append(chain, txid)

		// Get dependents and sort only if more than one
		dependents := dependedBy[txid]
		if len(dependents) > 1 {
			insertionSortByTime(dependents)
		}

		// Process each dependent
		for _, dep := range dependents {
			if _, ok := processed[dep]; ok {
				continue
			}

			// Quick check if all dependencies are processed
			canProcess := true
			for _, parentDep := range response[dep].Depends {
				if _, ok := processed[parentDep]; !ok {
					canProcess = false
					break
				}
			}

			if canProcess {
				processChain(dep)
			}
		}
	}

	// Process independent transactions
	if len(independentTxs) > 1 {
		insertionSortByTime(independentTxs)
	}

	for _, txid := range independentTxs {
		chain = chain[:0] // Reset chain slice without reallocating
		processChain(txid)
		if len(chain) > 0 {
			// Create new slice for this chain
			newChain := make([]string, len(chain))
			copy(newChain, chain)
			orderedGroups = append(orderedGroups, newChain)
		}
	}

	// Handle remaining transactions
	for txid := range response {
		if _, ok := processed[txid]; !ok {
			chain = chain[:0] // Reset chain slice without reallocating
			processChain(txid)
			if len(chain) > 0 {
				newChain := make([]string, len(chain))
				copy(newChain, chain)
				orderedGroups = append(orderedGroups, newChain)
			}
		}
	}

	return orderedGroups, nil
}
