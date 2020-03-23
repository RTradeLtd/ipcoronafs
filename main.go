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
			Name:  "scrape-all-location",
			Usage: "scrapes all location data",
			Action: func(c *cli.Context) error {
				defer cancel()
				tc, err := getFileClient(ctx, c.String("endpoint"), c.Bool("insecure"))
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
				tc, err := getFileClient(ctx, c.String("endpoint"), c.Bool("insecure"))
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
				tc, err := getFileClient(ctx, c.String("endpoint"), c.Bool("insecure"))
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
				tc, err := getFileClient(ctx, c.String("endpoint"), c.Bool("insecure"))
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

func getFileClient(ctx context.Context, endpoint string, insecure bool) (pb.FileAPIClient, error) {
	var (
		conn *grpc.ClientConn
		err  error
	)
	if insecure {
		conn, err = grpc.DialContext(ctx, defaultURL, grpc.WithInsecure())
	} else {
		conn, err = grpc.DialContext(ctx, defaultURL, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
	}
	if err != nil {
		return nil, err
	}
	return pb.NewFileAPIClient(conn), nil
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
