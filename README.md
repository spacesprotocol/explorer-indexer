# Overview

This repository contains an indexer for the spaces protocol explorer. 
The indexer retrieves block data from the bitcoin and spaces nodes and stores it into the postgresql database.

## Requirements
- Go v1.21 or higher
- PostgreSQL 14 or higher
- Docker v25.0.3 or higher (for containerized setups)
- Bitcoin Core node (for non-Docker setups)
- Spaces Protocol node (for non-Docker setups)

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
go build ./cmd/backfill
```

## Usage
The indexer provides two main executables:

### Sync Service
The primary service that indexes both Bitcoin and Spaces Protocol data:
```bash
./sync
```

Supports two sync modes:
- **Full Sync**: Indexes from the genesis block (slower but complete)
- **Fast Sync**: Starts from the Spaces Protocol activation block
  - Mainnet: Block 871222
  - Testnet4: Block 50000

### Backfill Service
Used to populate historical Bitcoin blocks when using Fast Sync mode:
```bash
./backfill
```
Note: Backfill only stores Bitcoin data, not Spaces Protocol data.

### Configuration
Configuration is handled through environment variables. Copy and modify the example configuration:
```bash
cp env.example .env
# Edit .env with your settings
```

## Development
There are three ways to set up the development environment:

### Complete docker regtest setup 

This setup is ideal for working on the [frontend part](https://github.com/spacesprotocol/explorer) as it provides a complete backend environment.

The docker setup in the `docker` folder provides:
- PostgreSQL database
- Automated database migrations
- Bitcoin node (regtest network)
- Spaced node
- Pre-configured spaces instances

Setup steps:
```bash
# Build the docker images
docker compose -f docker-regtest.yml build

# Start the services
docker compose -f docker-regtest.yml up
```

Docker data is stored in `regtest-data` directory.

### 2. PostgreSQL-only docker setup
If you're working on the indexer itself and want to manage the blockchain nodes separately, you can run just PostgreSQL in Docker:

```bash
# Start PostgreSQL container
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
4. Run the needed executable

### SQLC

To add create additional sql queries, it's advised to use SQLC. It generates idiomatic go code from the .sql types and queries. Query files are located in `sql/query`.

```
go install github.com/kyleconroy/sqlc/cmd/sqlc@latest
sqlc generate
```


