#! /bin/bash

while true; do
    CFG_PATH="config.yml"
    TOPIC="coronavirus-hash-broadcasts-latest-data"
    HASH=$(./ipcoronafs scrape-latest --with.timelines false | awk '{print $NF}')
    tex-cli --config "$CFG_PATH" client pubsub publish --topic "$TOPIC" --data "$HASH"
    echo "sleeping for 60 minutes"
    sleep 7200
; done

