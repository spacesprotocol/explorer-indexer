#!/bin/bash
# init-spaced.sh

spaced --block-index --chain=regtest --bitcoin-rpc-user=bitcoin --bitcoin-rpc-password=bitcoin --bitcoin-rpc-url=http://bitcoin:18443 --rpc-bind=0.0.0.0 &
SPACED_PID=$!

# Wait for both Bitcoin and Spaced to be ready
BLOCK_COUNT=$(PGPASSWORD=postgres psql -h db -U postgres -d postgres -t -c "SELECT COUNT(*) FROM blocks;" | tr -d ' ')
echo "latest block in the db is of height $BLOCK_COUNT"

if [ "$BLOCK_COUNT" -eq "0" ]; then
    echo "Empty database detected. Running initialization..."
    
    # Create and import wallet
    cat > /app/default.json << EOL
{
  "descriptor": "tr(tprv8ZgxMBicQKsPfHssgrtduF2bu21u81hhAnAzJoE4RnPnMsiD1i26ox6ysGgcXhTwupb9dtxKAb2AxBcP6xEWNXxBzstS7jhm3Uoxdj5rxF8/86'/1'/0'/0/*)",
  "blockheight": 0,
  "label": "default"
}
EOL

    echo "Importing wallet..."
    space-cli --chain regtest importwallet /app/default.json

    # Get address and mine initial blocks
    ADDRESS=$(space-cli --chain regtest getnewaddress)
    echo "Mining initial blocks to address: $ADDRESS"

    # Mine 101 blocks to make coins spendable
    curl -s -u bitcoin:bitcoin --data-binary "{\"jsonrpc\": \"1.0\", \"id\":\"mine\", \"method\": \"generatetoaddress\", \"params\": [101, \"$ADDRESS\"]}" -H 'content-type: text/plain;' http://bitcoin:18443/

    # Fixed array of asset names
    ASSETS=(
        "lemon" "watermelon" "artichoke" "ugli" "yuzu" "raspberry" "garlic" "nectarine" "nectarine" "vanilla"
        "broccoli" "cherry" "date" "elderberry" "fig" "grape" "honeydew" "kiwi" "mango" "orange"
        "papaya" "quince" "strawberry" "tangerine" "zucchini" "carrot" "daikon" "eggplant" "fennel" "horseradish"
        "iceberg" "jalapeno" "kale" "leek" "mushroom" "nutmeg" "okra" "parsley" "quinoa" "radish"
        "spinach" "tomato" "ulluco" "wasabi" "xigua" "yam" "zest" "apple" "banana" "plum"
    )

    # Open 50 assets with fixed names
    for i in {0..49}; do
        ASSET="${ASSETS[$i]}"
        echo "Opening asset: $ASSET"
        space-cli --chain regtest open "$ASSET" --fee-rate=1
        
        curl -s -u bitcoin:bitcoin --data-binary "{\"jsonrpc\": \"1.0\", \"id\":\"mine\", \"method\": \"generatetoaddress\", \"params\": [1, \"$ADDRESS\"]}" -H 'content-type: text/plain;' http://bitcoin:18443/
        sleep 1
    done

    # Mine 144 more blocks
    echo "Mining 144 additional blocks..."
    curl -s -u bitcoin:bitcoin --data-binary "{\"jsonrpc\": \"1.0\", \"id\":\"mine\", \"method\": \"generatetoaddress\", \"params\": [144, \"$ADDRESS\"]}" -H 'content-type: text/plain;' http://bitcoin:18443/

    # Send bid transactions for first 10 assets
    echo "Sending bid transactions for first 10 assets..."
    for i in {0..9}; do
        ASSET="${ASSETS[$i]}"
        echo "Bidding on asset: $ASSET"
        space-cli --chain regtest bid "$ASSET" 2000
        
        curl -s -u bitcoin:bitcoin --data-binary "{\"jsonrpc\": \"1.0\", \"id\":\"mine\", \"method\": \"generatetoaddress\", \"params\": [1, \"$ADDRESS\"]}" -H 'content-type: text/plain;' http://bitcoin:18443/
        sleep 1
    done

    echo "Initialization complete!"
else
    echo "Database already contains blocks. Skipping initialization."
fi

# Wait for the spaced process
wait $SPACED_PID
