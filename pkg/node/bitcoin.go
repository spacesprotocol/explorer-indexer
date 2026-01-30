package node

import (
	"context"
	"log"

	. "github.com/spacesprotocol/explorer-indexer/pkg/types"
)

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
		tx, err := client.GetTransaction(ctx, txid)
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

	size := len(response)
	log.Printf("found %d txs in node's mempool", size)
	orderedGroups := make([][]string, 0, size)

	for txid, info := range response {
		if len(info.Depends) == 0 {
			orderedGroups = append(orderedGroups, []string{txid})
		} else {
			group := make([]string, 0, len(info.Depends)+1)
			group = append(group, info.Depends...)
			group = append(group, txid)
			orderedGroups = append(orderedGroups, group)
		}
	}
	return orderedGroups, nil
}
