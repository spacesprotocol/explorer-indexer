#!/bin/bash
# init-wallet.sh

# Configuration
COMPOSE_FILE="docker-regtest.yml"
BLOCKS_TO_MINE=101  # Number of blocks to mine initially

# Create wallet file
create_wallet_file() {
    cat > default.json << EOL
{
  "descriptor": "tr(tprv8ZgxMBicQKsPfHssgrtduF2bu21u81hhAnAzJoE4RnPnMsiD1i26ox6ysGgcXhTwupb9dtxKAb2AxBcP6xEWNXxBzstS7jhm3Uoxdj5rxF8/86'/1'/0'/0/*)",
  "blockheight": 0,
  "label": "default"
}
EOL
    
    # Copy wallet file to spaced container
    docker compose -f $COMPOSE_FILE cp default.json spaced:/app/default.json
    rm default.json
}

# Function to check if database has any blocks
check_db_blocks() {
    BLOCK_COUNT=$(docker compose -f $COMPOSE_FILE exec -T db psql -U postgres -d postgres -t -c "SELECT COUNT(*) FROM blocks;" | tr -d ' ')
    echo $BLOCK_COUNT
}

# Function to import wallet to spaced
import_wallet() {
    echo "Importing wallet..."
    docker compose -f $COMPOSE_FILE exec -T spaced space-cli --chain regtest importwallet /app/default.json
}

# Function to get an address from the imported wallet
get_wallet_address() {
    docker compose -f $COMPOSE_FILE exec -T spaced space-cli --chain regtest getnewaddress
}

# Function to mine blocks
mine_blocks() {
    local ADDRESS=$1
    local COUNT=$2
    echo "Mining $COUNT blocks to address $ADDRESS..."
    docker compose -f $COMPOSE_FILE exec -T bitcoin bitcoin-cli -regtest \
        -rpcuser=bitcoin -rpcpassword=bitcoin \
        generatetoaddress $COUNT $ADDRESS
}

# Main execution
echo "Checking current blockchain state..."
BLOCK_COUNT=$(check_db_blocks)

if [ "$BLOCK_COUNT" -eq "0" ]; then
    echo "No blocks found in database. Initializing..."
    
    # Create and copy wallet file
    create_wallet_file
    
    # Import wallet
    import_wallet
    
    # Get address from the imported wallet
    echo "Getting wallet address..."
    ADDRESS=$(get_wallet_address)
    if [ -z "$ADDRESS" ]; then
        echo "Failed to get wallet address"
        exit 1
    fi
    
    echo "Got address: $ADDRESS"
    
    # Mine initial blocks
    mine_blocks "$ADDRESS" $BLOCKS_TO_MINE
    
    echo "Initialization complete!"
else
    echo "Found $BLOCK_COUNT blocks in database. Skipping initialization."
fi
