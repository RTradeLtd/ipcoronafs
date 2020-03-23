#! /bin/bash

while true; do
    CFG_PATH="config.yml"
    TOPIC="coronavirus-hash-broadcasts-latest-data"
    echo "scraping latest data with timelines"
    HASH=$(./ipcoronafs scrape-latest --with.timelines true | awk '{print $NF}')
    echo "broadcasting hash to pubsub topic $TOPIC"
    tex-cli --config "$CFG_PATH" client pubsub publish --topic "$TOPIC" --data "$HASH"
    echo "hash broadcasted: $HASH"
    echo "sleeping for 60 minutes"
    sleep 7200
done

