# Overview

This repository contains the indexer for the spaces protocol explorer. 
The indexer retrieves the blocks' data from the bitcoin and spaces nodes and stores it into the postgresql database.

## Install

```
go mod download
```

## Development

### Docker

There is a docker compose file for a complete backend setup which makes it easier to work on the [frontend
part](https://github.com/spacesprotocol/explorer) part. It is  located in `docker` folder which does the following:

- creates postgresql database
- runs needed migrations for the database
- runs bitcoin node for regtest network
- runs spaced node
- opens several dozens of spaces

Build the docker:
```
docker compose -f docker-regtest.yml build
```

Then run it:

```
docker compose -f docker-regtest.yml up
```

Docker data is stored in `regtest-data`.

### Manual setup

Run postgresql instance in docker:
```
docker-compose up
```

### Migrations

[Goose](https://github.com/pressly/goose) is used for migrations. Migrations are located in `sql/schema`.

```
. ./env.example
goose up
```

### Blockchain nodes

Run bitcoind and spaced nodes, their URIs should be stored as environment variables which are later used by the sync
service.

### Sync

Sync is a go service which stores the blockchain data.

```
. ./env.example
go run cmd/sync/*
```

Now you should have a working service.

### SQLC

To add create additional sql queries, it's advised to use SQLC. It generates idiomatic go code from the .sql types and queries. Query files are located in `sql/query`.

```
go install github.com/kyleconroy/sqlc/cmd/sqlc@latest
sqlc generate
```


