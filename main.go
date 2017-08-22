package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
)

func main() {
	elasticURL := flag.String("url", "http://localhost:9200", "elastic search api url")
	index := flag.String("index", "pastebin", "pastebin")
	flag.Parse()

	e := &ElasticStorageModule{
		index:      *index,
		elasticURL: *elasticURL,
	}

	go FetchWorker(e)
	go QueryWorker()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		stopFetchWorker = true
		stopQueryWorker = true
		log.Print("Shutting down... Please Wait")
	}()

	log.Print("Scraper running")

	done := make(chan bool)
	go func() {
		<-queryWorkerStopped
		done <- true
	}()
	go func() {
		<-fetchWorkerStopped
		done <- true
	}()

	for i := 0; i < 2; i++ {
		<-done
	}

	log.Print("Scraper exiting")
}
