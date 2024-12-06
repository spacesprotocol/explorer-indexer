# Dockerfile.migrations
FROM golang:1.14.3-alpine
RUN apk add --no-cache gcc g++ curl

# Install goose
# RUN go get github.com/pressly/goose/cmd/goose
RUN curl -fsSL \
    https://raw.githubusercontent.com/pressly/goose/master/install.sh |\
    sh

WORKDIR /src
COPY sql/schema /src/sql/schema

ENTRYPOINT ["/bin/sh", "-c", "goose -dir sql/schema postgres $POSTGRES_URI up"]
