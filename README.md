# Overview

This repository contains the indexer for the spaces protocol explorer. 
The indexer retrieves the blocks' data from the bitcoin and spaced nodes and stored into the postgresql database.
Later it can be used to provide an API rest service.

## Install

```
go mod download
```

## Migrations

[Goose](https://github.com/pressly/goose) is used for migrations. 

```
. ./env
goose up
```

## SQLC

Generates idiomatic go code from the .sql types and queries.

```
go install github.com/kyleconroy/sqlc/cmd/sqlc@latest
sqlc generate
```


## Local setup

Run postgresql instance in docker:
```
docker-compose up
```


Run: 

```
go run cmd/sync/*
```
