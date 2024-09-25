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

func (client *SpacesClient) GetSpace(ctx context.Context, space string) (*Space, error) {
	if len(space) > 0 && space[0] != '@' {
		space = "@" + space
	}
	spaceInfo := new(Space)
	err := client.Rpc(ctx, "getspace", []interface{}{space}, spaceInfo)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return spaceInfo, err
}

func (client *SpacesClient) GetSpaceOwner(ctx context.Context, space string) ([]byte, error) {
	if len(space) > 0 && space[0] != '@' {
		space = "@" + space
	}
	var spaceOwner []byte
	err := client.Rpc(ctx, "getspaceowner", []interface{}{space}, spaceOwner)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return spaceOwner, err
}

func (client *SpacesClient) GetSpaceOut(ctx context.Context, outpoint string) (*Space, error) {
	space := new(Space)
	err := client.Rpc(ctx, "getspaceout", []interface{}{outpoint}, space)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return space, err
}

func (client *SpacesClient) GetRollOut(ctx context.Context, number int) (*[]Space, error) {
	var spaces []Space
	err := client.Rpc(ctx, "getrollout", []interface{}{number}, spaces)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return &spaces, err
}

func (client *SpacesClient) GetBlockData(ctx context.Context, blockHash string) (*SpacesBlock, error) {
	txs := new(SpacesBlock)
	err := client.Rpc(ctx, "getblockdata", []interface{}{blockHash}, txs)
	if err != nil {
		log.Print(err)
		return nil, err
	}
	return txs, err
}
