package main

import (
	"log"
	"strings"
	"testing"
)

func TestQueryPastes(t *testing.T) {
	pastes, err := QueryPastes()
	if err != nil {
		t.Fail()
	}
	if pastes == nil {
		t.Fail()
	}
}

func TestFetchPaste(t *testing.T) {
	p := PasteMeta{ScrapeURL: "https://pastebin.com/api_scrape_item.php?i=bDDWiUHd"}
	data, err := FetchPaste(p)
	if err != nil {
		t.Fail()
	}
	if !strings.Contains(data, "9jj1P/ycn7YtVofT") {
		t.Fail()
	}
}

func TestQueryWorker(t *testing.T) {
	go QueryWorker()
	paste := <-chanPasteMeta
	log.Print(paste)
	stopQueryWorker = true
	<-queryWorkerStopped
}

type TestStorageModule struct {
}

func (t *TestStorageModule) StorePaste(paste PasteFull) {
	log.Printf("Paste: %+v", paste.Data[0:0x10])
}

func (t *TestStorageModule) Initialize() error {
	log.Printf("Storage module initialized")
	return nil
}

func (t *TestStorageModule) Destroy() error {
	log.Printf("Storage module destroyed")
	return nil
}

func TestFetchWorker(t *testing.T) {
	tsm := &TestStorageModule{}
	go FetchWorker(tsm)
	go QueryWorker()
	stopFetchWorker = true
	stopQueryWorker = true
	<-fetchWorkerStopped
	<-queryWorkerStopped
}
