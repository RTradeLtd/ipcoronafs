# ipcoronafs

`ipcoronafs` is a tool to scrape information from [coronavirus api tracker](https://github.com/ExpDev07/coronavirus-tracker-api) using [go-corona](https://github.com/itsksaurabh/go-corona) and storing it on IPFS through TemporalX. It expects a locally running TemporalX server, but can be be configured to use a remote one..

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
