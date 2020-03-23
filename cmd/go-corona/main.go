package main

import (
	"context"
	"fmt"
	"log"

	gocorona "github.com/itsksaurabh/go-corona"
)

func main() {
	// client for accessing different endpoints of the API
	client := gocorona.Client{}
	ctx := context.Background()

	// GetLatestData returns total amonut confirmed cases, deaths, and recoveries.
	data, err := client.GetLatestData(ctx)
	if err != nil {
		log.Fatal("request failed:", err)
	}
	fmt.Println(data)
}
