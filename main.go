package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"
)

func main() {
	var wg sync.WaitGroup

	elasticURL := flag.String("url", "http://localhost:9200", "elastic search api url")
	index := flag.String("index", "pastebin", "pastebin")
	dailyIndex := flag.Bool("daily", false, time.Now().Format("use daily indexes, e.g. pastebin-2006-1-2"))
	APIServer := flag.String("server", "pastebin.com", "Pastebin scraping API url")
	flag.Parse()

	// make elastic storage module
	e := &ElasticStorageModule{
		index:        *index,
		dailyIndexes: *dailyIndex,
		elasticURL:   *elasticURL,
	}

	// make and start fetch worker
	fw := FetchWorker{
		OutModules:             []StorageModule{e},
		PastebinURLReplacement: *APIServer,
	}
	fw.Run(&wg)

	// make and start query worker
	qw := QueryWorker{
		APIServer: *APIServer,
		SleepTime: 30000 * time.Millisecond,
	}
	qw.Run(&wg)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		log.Print("Shutting down... Please Wait")
		fw.Stop(&wg)
		qw.Stop(&wg)
	}()

	wg.Wait()
	log.Print("Shutdown complete")
}
