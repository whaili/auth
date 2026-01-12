#!/bin/bash

export PORT=8081
export MONGO_URI=mongodb://admin:123456@localhost:27017
export ACCOUNT_FETCHER_MODE=local
export QINIU_UID_MAPPER_MODE=simple
export QINIU_UID_AUTO_CREATE=false
export HMAC_TIMESTAMP_TOLERANCE=15m

echo "Starting Bearer Token Service on port 8081..."
./bin/tokenserv
