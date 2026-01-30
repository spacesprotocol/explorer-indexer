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

func (client *SpacesClient) GetBlockMeta(ctx context.Context, blockIdentifier interface{}) (*SpacesBlock, error) {
	txs := new(SpacesBlock)
	err := client.Rpc(ctx, "getblockmeta", []interface{}{blockIdentifier}, txs)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return txs, err
}

func (client *SpacesClient) GetPtrBlockMeta(ctx context.Context, blockIdentifier any) (*PtrBlock, error) {
	ptrBlock := new(PtrBlock)
	err := client.Rpc(ctx, "getptrblockmeta", []any{blockIdentifier}, ptrBlock)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return ptrBlock, err
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

func (client *SpacesClient) VerifyListing(ctx context.Context, listing Listing) error {
	qwer := new(struct{})
	err := client.Rpc(ctx, "verifylisting", []interface{}{listing}, &qwer)
	if err != nil {
		return err
	}
	return nil
}

func (client *SpacesClient) GetRootAnchors(ctx context.Context) ([]*RootAnchor, error) {
	rootAnchors := make([]*RootAnchor, 0)
	err := client.Rpc(ctx, "getrootanchors", []interface{}{}, &rootAnchors)
	if err != nil {
		return nil, err
	}
	return rootAnchors, nil
}
