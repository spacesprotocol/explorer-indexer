#!/bin/bash
# mine-blocks.sh

# Default number of blocks to mine
BLOCKS=${1:-1}

# Create and load wallet if it doesn't exist
echo "Creating/loading wallet..."
docker compose exec -T bitcoin bitcoin-cli -regtest -rpcuser=bitcoin -rpcpassword=bitcoin createwallet "regtest" >/dev/null 2>&1 || \
docker compose exec -T bitcoin bitcoin-cli -regtest -rpcuser=bitcoin -rpcpassword=bitcoin loadwallet "regtest" >/dev/null 2>&1

# Get a new address to mine to
# ADDRESS=$(docker compose exec -T bitcoin bitcoin-cli -regtest -rpcuser=bitcoin -rpcpassword=bitcoin getnewaddress)
ADDRESS="bcrt1px790k8vzn5v8glt7tkmtj85u4a3g9k0vzz557hvm35rgzgqhfl4smt0dg5"
echo "Mining to address: $ADDRESS"

# Generate blocks
echo "Mining $BLOCKS blocks..."
docker compose exec -T bitcoin bitcoin-cli -regtest -rpcuser=bitcoin -rpcpassword=bitcoin generatetoaddress $BLOCKS "$ADDRESS"

# Show blockchain info
echo -e "\nBlockchain Info:"
docker compose exec -T bitcoin bitcoin-cli -regtest -rpcuser=bitcoin -rpcpassword=bitcoin getblockchaininfo
