package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"

	pb "github.com/RTradeLtd/TxPB/v3/go"
	gocorona "github.com/itsksaurabh/go-corona"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	defaultURL = "127.0.0.1:9090"
	insecure   = true
)

const (
	allLocationDataTopic    = "coronavirus-all-location-data-topic"
	latestLocationDataTopic = "coronavirus-latest-location-data-topic"
)

func newApp(ctx context.Context, cancel context.CancelFunc) *cli.App {
	app := cli.NewApp()
	app.Name = "ipcoronafs"
	app.Usage = "scrapes coronavirus tracker information and stores on IPFS"
	app.Description = "using TemporalX we can store massive data sets, and what better client to use for tracking coronavirus case information"
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "insecure",
			Usage: "enable insecure access to TemporalX apis",
			Value: true,
		},
		&cli.StringFlag{
			Name:  "endpoint",
			Usage: "temporalx endpoint to connect to",
			Value: "127.0.0.1:9090",
		},
	}
	app.Commands = cli.Commands{
		&cli.Command{
			Name:  "service",
			Usage: "run the temporalx ipcoronafs service",
			Action: func(c *cli.Context) error {
				allRebroadCaster := make(chan string, 1)
				defer close(allRebroadCaster)
				latestRebroadcaster := make(chan string, 1)
				defer close(latestRebroadcaster)
				defer cancel()
				conn, err := getConn(ctx, c.String("endpoint"), c.Bool("insecure"))
				defer conn.Close()
				ps, err := getPubSubClient(ctx, conn)
				if err != nil {
					return err
				}
				pubsub, err := ps.PubSub(ctx)
				if err != nil {
					return err
				}
				defer pubsub.CloseSend()
				fc, err := getFileClient(ctx, conn)
				if err != nil {
					return err
				}
				handleLatest := func() error {
					// client for accessing different endpoints of the API
					gorona := gocorona.Client{}
					fmt.Println("getting all latest data")
					locations, err := gorona.GetLatestData(ctx)
					if err != nil {
						return err
					}
					data, err := json.Marshal(locations)
					if err != nil {
						return err
					}
					fmt.Println("adding latest data to ipfs")
					hash, err := uploadFile(fc, bytes.NewReader(data))
					if err != nil {
						return err
					}
					fmt.Println("latest data hash: ", hash)
					if err := pubsub.Send(&pb.PubSubRequest{
						RequestType: pb.PSREQTYPE_PS_PUBLISH,
						Topics:      []string{latestLocationDataTopic},
						Data:        []byte(hash),
					}); err != nil {
						log.Println("ERROR: failed to send latest data via pubsub")
					}
					latestRebroadcaster <- hash
					fmt.Println("sent pubsub data")
					return nil
				}
				handleAll := func() error {
					// client for accessing different endpoints of the API
					gorona := gocorona.Client{}
					fmt.Println("getting all location data")
					locations, err := gorona.GetAllLocationData(ctx, c.Bool("with.timelines"))
					if err != nil {
						return err
					}
					data, err := json.Marshal(locations)
					if err != nil {
						return err
					}
					fmt.Println("adding location data to ipfs")
					hash, err := uploadFile(fc, bytes.NewReader(data))
					if err != nil {
						return err
					}
					fmt.Println("all location data hash: ", hash)
					if err := pubsub.Send(&pb.PubSubRequest{
						RequestType: pb.PSREQTYPE_PS_PUBLISH,
						Topics:      []string{allLocationDataTopic},
						Data:        []byte(hash),
					}); err != nil {
						log.Println("ERROR: failed to send latest data via pubsub")
					}
					allRebroadCaster <- hash
					return nil
				}
				allTicker := time.NewTicker(time.Hour)
				defer allTicker.Stop()
				latestTicker := time.NewTicker(time.Hour)
				defer latestTicker.Stop()
				// run a latest fetch
				if err := handleLatest(); err != nil {
					log.Println("ERROR: failed to get latest: ", err)
				}
				// run an all fetch
				if err := handleAll(); err != nil {
					log.Println("ERROR: failed to get all: ", err)
				}
				rebroadcaster := time.NewTicker(time.Minute)
				defer rebroadcaster.Stop()
				var (
					lastCancel   context.CancelFunc
					lastContext  context.Context
					lastCancel2  context.CancelFunc
					lastContext2 context.Context
					lock         sync.Mutex
				)
				for {
					select {
					case <-ctx.Done():
						return nil
					case <-allTicker.C:
						go func() {
							if err := handleAll(); err != nil {
								log.Println("ERROR: failed to get all: ", err)
							}
						}()
					case <-latestTicker.C:
						go func() {
							if err := handleLatest(); err != nil {
								log.Println("ERROR: failed to get latest: ", err)
							}
						}()
					case hash := <-allRebroadCaster:
						fmt.Println("handling all rebroadcast")
						lock.Lock()
						if lastCancel != nil {
							lastCancel()
						}
						lastContext, lastCancel = context.WithCancel(ctx)
						lock.Unlock()
						go func() {
							ticker := time.NewTicker(time.Minute)
							defer ticker.Stop()
							for {
								select {
								case <-lastContext.Done():
									return
								case <-ticker.C:
									fmt.Println("sending rebroadcast")
									if err := pubsub.Send(&pb.PubSubRequest{
										RequestType: pb.PSREQTYPE_PS_PUBLISH,
										Topics:      []string{allLocationDataTopic},
										Data:        []byte(hash),
									}); err != nil {
										log.Println("ERROR: failed to send latest data via pubsub")
									}
								}
							}
						}()
						fmt.Println("sending all location data hash through pubsub")

					case hash := <-latestRebroadcaster:
						fmt.Println("handling latest rebroadcaster")
						lock.Lock()
						if lastCancel2 != nil {
							lastCancel2()
						}
						lastContext2, lastCancel2 = context.WithCancel(ctx)
						lock.Unlock()
						go func() {
							ticker := time.NewTicker(time.Minute)
							defer ticker.Stop()
							for {
								select {
								case <-lastContext2.Done():
									return
								case <-ticker.C:
									fmt.Println("sending rebroadcast")
									if err := pubsub.Send(&pb.PubSubRequest{
										RequestType: pb.PSREQTYPE_PS_PUBLISH,
										Topics:      []string{allLocationDataTopic},
										Data:        []byte(hash),
									}); err != nil {
										log.Println("ERROR: failed to send latest data via pubsub")
									}
								}
							}
						}()
						fmt.Println("sending all location data hash through pubsub")

					}

				}
			},
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "with.timelines",
					Usage: "enable timeline scraping",
					Value: false,
				},
			},
		},
		&cli.Command{
			Name:  "scrape-all-location",
			Usage: "scrapes all location data",
			Action: func(c *cli.Context) error {
				defer cancel()
				conn, err := getConn(ctx, c.String("endpoint"), c.Bool("insecure"))
				defer conn.Close()
				tc, err := getFileClient(ctx, conn)
				if err != nil {
					return err
				}
				// client for accessing different endpoints of the API
				gorona := gocorona.Client{}
				locations, err := gorona.GetAllLocationData(ctx, c.Bool("with.timelines"))
				if err != nil {
					return err
				}
				data, err := json.Marshal(locations)
				if err != nil {
					return err
				}
				hash, err := uploadFile(tc, bytes.NewReader(data))
				if err != nil {
					return err
				}
				fmt.Println("locations hash: ", hash)
				return nil
			},
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "with.timelines",
					Usage: "enable timeline scraping",
					Value: false,
				},
			},
		},
		&cli.Command{
			Name:  "scrape-latest",
			Usage: "scrapes latest set of data",
			Action: func(c *cli.Context) error {
				defer cancel()
				conn, err := getConn(ctx, c.String("endpoint"), c.Bool("insecure"))
				defer conn.Close()
				tc, err := getFileClient(ctx, conn)
				if err != nil {
					return err
				}
				// client for accessing different endpoints of the API
				gorona := gocorona.Client{}
				latest, err := gorona.GetLatestData(ctx)
				if err != nil {
					return err
				}
				data, err := json.Marshal(latest)
				if err != nil {
					return err
				}
				hash, err := uploadFile(tc, bytes.NewReader(data))
				if err != nil {
					return err
				}
				fmt.Println("latest data hash: ", hash)
				return nil
			},
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "with.timelines",
					Usage: "enable timeline scraping",
					Value: false,
				},
			},
		},
		&cli.Command{
			Name:  "scrape-data-by-location-id",
			Usage: "scrapes latest set of data",
			Action: func(c *cli.Context) error {
				defer cancel()
				conn, err := getConn(ctx, c.String("endpoint"), c.Bool("insecure"))
				defer conn.Close()
				tc, err := getFileClient(ctx, conn)
				if err != nil {
					return err
				}
				// client for accessing different endpoints of the API
				gorona := gocorona.Client{}
				latest, err := gorona.GetDataByLocationID(ctx, c.Int("location.id"), c.Bool("with.timelines"))
				if err != nil {
					return err
				}
				data, err := json.Marshal(latest)
				if err != nil {
					return err
				}
				hash, err := uploadFile(tc, bytes.NewReader(data))
				if err != nil {
					return err
				}
				fmt.Println("latest data hash: ", hash)
				return nil
			},
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "with.timelines",
					Usage: "enable timeline scraping",
					Value: false,
				},
				&cli.IntFlag{
					Name:  "location.id",
					Usage: "the location id to use",
				},
			},
		},
		&cli.Command{
			Name:  "scrape-data-by-country-code",
			Usage: "scrapes latest set of data going by country code",
			Action: func(c *cli.Context) error {
				defer cancel()
				conn, err := getConn(ctx, c.String("endpoint"), c.Bool("insecure"))
				defer conn.Close()
				tc, err := getFileClient(ctx, conn)
				if err != nil {
					return err
				}
				// client for accessing different endpoints of the API
				gorona := gocorona.Client{}
				latest, err := gorona.GetDataByCountryCode(ctx, c.String("country.code"), c.Bool("with.timelines"))
				if err != nil {
					return err
				}
				data, err := json.Marshal(latest)
				if err != nil {
					return err
				}
				hash, err := uploadFile(tc, bytes.NewReader(data))
				if err != nil {
					return err
				}
				fmt.Println("latest data hash: ", hash)
				return nil
			},
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:  "with.timelines",
					Usage: "enable timeline scraping",
					Value: false,
				},
				&cli.StringFlag{
					Name:  "country.code",
					Usage: "the location id to use",
				},
			},
		},
	}
	return app
}

func getConn(ctx context.Context, endpoint string, insecure bool) (*grpc.ClientConn, error) {
	if insecure {
		return grpc.DialContext(ctx, defaultURL, grpc.WithInsecure())
	}
	return grpc.DialContext(ctx, defaultURL, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
}

func getFileClient(ctx context.Context, conn *grpc.ClientConn) (pb.FileAPIClient, error) {
	return pb.NewFileAPIClient(conn), nil
}
func getPubSubClient(ctx context.Context, conn *grpc.ClientConn) (pb.PubSubAPIClient, error) {
	return pb.NewPubSubAPIClient(conn), nil
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app := newApp(ctx, cancel)
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func uploadFile(nc pb.FileAPIClient, data io.Reader) (string, error) {
	stream, err := nc.UploadFile(context.Background())
	if err != nil {
		return "", err
	}
	// declare file options
	if err := stream.Send(&pb.UploadRequest{Options: &pb.UploadOptions{MultiHash: "sha2-256", Chunker: "size-1"}}); err != nil {
		return "", err
	}
	// upload file - chunked at 5mb each
	buf := make([]byte, 4194294)
	for {
		n, err := data.Read(buf)
		if err != nil && err == io.EOF {
			// only break if we haven't read any bytes, otherwise exit
			if n == 0 {
				break
			}
		} else if err != nil && err != io.EOF {
			return "", err
		}
		if err := stream.Send(&pb.UploadRequest{Blob: &pb.Blob{Content: buf[:n]}}); err != nil {
			return "", err
		}
	}
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return "", err
	}
	return resp.GetHash(), nil
}
