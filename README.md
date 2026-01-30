# Overview

This repository contains an indexer for the spaces protocol explorer. 
The indexer retrieves block data from the bitcoin and spaces nodes and stores it into the postgresql database.

## Operation

It stores both bitcoin and spaces data, for the bitcoin data it only stores tx hashes and number of inputs and outputs
along with their values (no scriptsig/scriptpubkey/witness data). 

It is capable of detecting chain reorganizations, moreover it stores orphaned txs. Also it syncs the local bitcoin node mempool and parses corresponding spaces events (however
does not store space pointers mempool events, as the needed spaced API is not ready as of time of writing).


It has several modes of operation, the main one comes with an executable `sync` from `cmd/sync`, which does the sync.
It has 'fast' sync which starts not from the genesis block, but from FAST_SYNC_BLOCK_HEIGHT, also will sync spaces data
only from ACTIVATION_BLOCK_HEIGHT. See additional environment variables in env.example.

Another executable is `populate` which is used only to populate spaces data, which is useful when a new non-backward
compatible feature is added: one can delete spaces-related tables completely and make a fresh sync which is considerably quick.

### Database Design Note

Since the indexer stores both orphaned blocks and mempool transactions, there are "virtual" blocks in the database:
- Mempool transactions use a special block hash: `deadbeefdeadbeef...` 
- Orphaned blocks are kept with `orphan = true` flag

Because of this, a transaction is uniquely identified by the pair `(block_hash, txid)`, not just by `txid` alone. The same transaction may appear multiple times with different block hashes (e.g., once in mempool, once in a confirmed block, or in competing chain tips).

## Reusable Packages

The `pkg/node` package provides Go wrappers for Bitcoin Core and Spaces daemon RPC APIs that can be imported by other projects:

```go
import "github.com/spacesprotocol/explorer-indexer/pkg/node"

// Bitcoin Core RPC client
bc := node.BitcoinClient{Client: node.NewClient(uri, user, password)}
block, err := bc.GetBlock(ctx, blockHash)

// Spaces daemon RPC client
sc := node.SpacesClient{Client: node.NewClient(uri, user, password)}
meta, err := sc.GetBlockMeta(ctx, blockHash)
```

This package is used by other projects such as [spaces marketplace](https://spaces.market)

## Requirements
- Go v1.21 or higher
- PostgreSQL 16 or higher
- Bitcoin Core node 
- Spaces protocol daemon 

## Installation
1. Clone the repository
```bash
git clone https://github.com/spacesprotocol/explorer-indexer
cd explorer-indexer
```

2. Install dependencies
```bash
go mod download
```

3. Build the executables
```bash
go build ./cmd/sync
go build ./cmd/populate
```
## Running 

### Sync Service
The primary service that indexes both bitcoin and spaces protocol data:
```bash
./sync
```

### Configuration
Configuration is handled through environment variables. Copy and modify the example configuration:
```bash
cp env.example .env
# Edit .env with your settings
```

## Development

### Dockerized setup
Setup steps:
```bash
# Build the docker images
docker compose -f docker-regtest.yml build

# Start the services
docker compose -f docker-regtest.yml up
```

Docker data is stored in `regtest-data` directory.

### Dockerfile for PostgreSQL
If you're working on the indexer itself and want to manage the blockchain nodes separately, you can run just PostgreSQL in docker:

```bash
docker-compose up
```

#### Migrations 

You will also need to run migrations for the database, they are managed with [Goose](https://github.com/pressly/goose). Migrations are located in `sql/schema`.

```
. ./env.example
goose up
go run cmd/sync/*
```


### Manual Setup

For complete control over your environment, you can:
1. Run PostgreSQL directly on your system and run migrations
2. Set up Bitcoin and Spaces nodes manually
3. Configure the environment variables to point to your services
4. Run the sync process

### SQLC

To create additional sql queries, it's advised to use SQLC. It generates idiomatic go code from the .sql types and queries. Query files are located in `sql/query`.

```
go install github.com/kyleconroy/sqlc/cmd/sqlc@latest
sqlc generate
```

## Notes

Previously the indexer indexed all inputs/outputs and the whole blockchain data, there might be some remnants of it.


## License 

MIT
