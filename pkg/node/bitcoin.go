package node

import (
	"context"
	"log"

	. "github.com/spacesprotocol/explorer-backend/pkg/types"
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

// func (client *BitcoinClient) GetSpace(space string) (*Space, error) {
// 	if len(space) > 0 && space[0] != '@' {
// 		space = "@" + space
// 	}
// 	ctx := context.Background()
// 	spaceInfo := new(Space)
// 	err := client.Rpc(ctx, "getspace", []interface{}{space}, spaceInfo)
// 	if err != nil {
// 		log.Print(err)
// 		return nil, err
// 	}
// 	return spaceInfo, err
// }
//
// func (client *BitcoinClient) GetSpaceOwner(space string) ([]byte, error) {
// 	if len(space) > 0 && space[0] != '@' {
// 		space = "@" + space
// 	}
// 	ctx := context.Background()
// 	var spaceOwner []byte
// 	err := client.Rpc(ctx, "getspaceowner", []interface{}{space}, spaceOwner)
// 	if err != nil {
// 		log.Print(err)
// 		return nil, err
// 	}
// 	return spaceOwner, err
// }
