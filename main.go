package main

import (
	"log"
	"os"
	"os/signal"
)

func main() {
	e := &ElasticStorageModule{
		index:      "pastebin",
		elasticURL: "http://elk.lab",
	}

	go FetchWorker(e)
	go QueryWorker()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		stopFetchWorker = true
		stopQueryWorker = true
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
