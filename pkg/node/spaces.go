package node

import (
	"context"
	"log"
)

type SpacesClient struct {
	*Client
}

func (client *SpacesClient) GetServerInfo(ctx context.Context) (*ServerInfo, error) {
	serverInfo := new(ServerInfo)
	err := client.Rpc(ctx, "getserverinfo", nil, serverInfo)
	if err != nil {
		return nil, err
	}
	return serverInfo, err
}

func (client *SpacesClient) GetRollOut(ctx context.Context, number int) (*[]RollOutSpace, error) {
	var rollout []RollOutSpace
	err := client.Rpc(ctx, "getrollout", []interface{}{number}, &rollout)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return &rollout, err
}

func (client *SpacesClient) GetBlockMeta(ctx context.Context, blockHash string) (*SpacesBlock, error) {
	txs := new(SpacesBlock)
	err := client.Rpc(ctx, "getblockmeta", []interface{}{blockHash}, txs)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return txs, err
}

func (client *SpacesClient) GetTxMeta(ctx context.Context, txId string) (*MetaTransaction, error) {
	metaTx := new(MetaTransaction)
	err := client.Rpc(ctx, "gettxmeta", []interface{}{txId}, metaTx)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return metaTx, err
}

func (client *SpacesClient) CheckPackage(ctx context.Context, txHexes []string) ([]*MetaTransaction, error) {
	metaTxs := make([]*MetaTransaction, 0)
	err := client.Rpc(ctx, "checkpackage", []interface{}{txHexes}, &metaTxs)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return metaTxs, err
}
