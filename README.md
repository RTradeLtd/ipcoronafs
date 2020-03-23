# ipcoronafs

`ipcoronafs` is a tool to scrape information from [coronavirus api tracker](https://github.com/ExpDev07/coronavirus-tracker-api) using [go-corona](https://github.com/itsksaurabh/go-corona) and storing it on IPFS through TemporalX. It expects a locally running TemporalX server, but can be be configured to use a remote one..

# Overview

For an overview of this see [medium](https://medium.com/temporal-cloud/real-time-covid-19-sars-cov-2-outbreak-stats-over-libp2p-ipfs-75972c9afa7)

# Workflow

* Every 60 minutes we use the go-corona client to fetch the "latest location data" and "all location data" sources.
* These are then added to IPFS and the hash is broadcast over libp2p pubsub topics
* Every minute we then rebroadcast the lastest known hash, which gets updated every 60 minutes
* Every 12 hours or so update the DNSLink record

# Real Time Information

We are broadcasting updates to this in somewhat real-time. You can connect to two different pubsub topics.

To retrieve updates for all known location outbreak information do:

```
1) ipfs swarm connect /ip4/206.116.153.42/tcp/4005/p2p/12D3KooWLrKQEE5NfA3vmEXfRFWKzNjq7DuwVS9CsZ8ooPe7sZFM

2) ipfs swarm connect /ip4/206.116.153.42/tcp/4004/ipfs/QmePr8gxUswSsD7anQCm8P1F599CrmK2Wze1DjoN8LaLAx

3) ipfs pubsub sub coronavirus-all-location-data-topic and then within about a minute you should start seeing the data coming through.

4) ipfs pin add <hash from pubsub message> and you'll then be pinning the data locally
```

To retrieve updates for latest ooutbreak informatio updates do

```
1) ipfs swarm connect /ip4/206.116.153.42/tcp/4005/p2p/12D3KooWLrKQEE5NfA3vmEXfRFWKzNjq7DuwVS9CsZ8ooPe7sZFM

2) ipfs swarm connect /ip4/206.116.153.42/tcp/4004/ipfs/QmePr8gxUswSsD7anQCm8P1F599CrmK2Wze1DjoN8LaLAx

3) ipfs pubsub sub coronavirus-latest-location-data-topic and then within about a minute you should start seeing the data coming through.

4) ipfs pin add <hash from pubsub message> and you'll then be pinning the data locally
```
