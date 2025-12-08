package main

import (
	"context"
	"log"
	"os"
	"time"
)

func main() {
	siteID := os.Getenv("SITE_ID")
	log.Printf("Starting site agent for site: %s", siteID)
	
	for {
		log.Println("Sync cycle...")
		time.Sleep(30 * time.Second)
	}
}
