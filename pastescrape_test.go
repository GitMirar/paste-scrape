package main

import (
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	httpmock "gopkg.in/jarcoal/httpmock.v1"
)

const testMetaJSON = `[{
		"scrape_url": "https://pastebin.com/api_scrape_item.php?i=bRuV7pSD",
		"full_url": "https://pastebin.com/bRuV7pSD",
		"date": "1503599704",
		"key": "bRuV7pSD",
		"size": "38042",
		"expire": "0",
		"title": "",
		"syntax": "text",
		"user": ""
}]`

func TestQueryPastes(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://pastebin.com/api_scraping.php",
		httpmock.NewStringResponder(200, testMetaJSON))

	pastes, err := QueryPastes("pastebin.com")
	if err != nil {
		t.Fail()
	}
	if pastes == nil {
		t.Fail()
	}
}

func TestFetchPaste(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://pastebin.com/api_scrape_item.php?i=bDDWiUHd",
		httpmock.NewStringResponder(200, `foo bar`))

	p := PasteMeta{
		ScrapeURL: "https://pastebin.com/api_scrape_item.php?i=bDDWiUHd",
	}
	data, err := FetchPaste(p)
	if err != nil {
		t.Fail()
	}
	if !strings.Contains(data, "foo bar") {
		t.Fail()
	}
}

func TestQueryWorker(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://pastebin.com/api_scraping.php",
		httpmock.NewStringResponder(200, testMetaJSON))

	httpmock.RegisterResponder("GET", "https://pastebin.com/api_scrape_item.php?i=bRuV7pSD",
		httpmock.NewStringResponder(200, `foo bar baz`))

	var wg sync.WaitGroup
	qw := QueryWorker{
		APIServer: "pastebin.com",
		SleepTime: 5 * time.Second,
	}
	qw.Run(&wg)
	<-chanPasteMeta
	qw.Stop(&wg)
	wg.Wait()
}

type TestStorageModule struct {
	Collected []string
}

func (t *TestStorageModule) StorePaste(paste PasteFull) {
	log.Printf("Paste: %+v", paste.Data)
	t.Collected = append(t.Collected, string(paste.Data))
}

func (t *TestStorageModule) Initialize() error {
	log.Printf("Storage module initialized")
	t.Collected = make([]string, 0)
	return nil
}

func (t *TestStorageModule) Destroy() error {
	log.Printf("Storage module destroyed")
	return nil
}

func (t *TestStorageModule) Check(paste PasteMeta) bool {
	return true
}

func TestFetchWorker(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	httpmock.RegisterResponder("GET", "https://pastebin.com/api_scraping.php",
		httpmock.NewStringResponder(200, testMetaJSON))

	httpmock.RegisterResponder("GET", "https://pastebin.com/api_scrape_item.php?i=bRuV7pSD",
		httpmock.NewStringResponder(200, `foo bar baz`))

	tsm := &TestStorageModule{}
	var wg sync.WaitGroup
	fw := FetchWorker{
		OutModules: []StorageModule{tsm},
	}
	fw.Run(&wg)
	qw := QueryWorker{
		APIServer: "pastebin.com",
		SleepTime: 5 * time.Second,
	}
	qw.Run(&wg)
	time.Sleep(2 * time.Second)
	fw.Stop(&wg)
	qw.Stop(&wg)
	wg.Wait()

	if len(tsm.Collected) != 1 {
		t.Fatalf("unexpected length of collected results, 1 != %d",
			len(tsm.Collected))
	}
	if tsm.Collected[0] != "foo bar baz" {
		t.Fatalf("unexpected value of collected results, fo bar baz != %s",
			tsm.Collected[0])
	}
}
