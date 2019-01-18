package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// PasteMeta is a set of descriptive information on a paste.
type PasteMeta struct {
	ScrapeURL string `json:"scrape_url"`
	FullURL   string `json:"full_url"`
	Date      string `json:"date"`
	Key       string `json:"key"`
	Size      string `json:"size"`
	Expire    string `json:"expire"`
	Title     string `json:"title"`
	Syntax    string `json:"syntax"`
	User      string `json:"user"`
}

// PasteFull extends PasteMeta by the actual content.
type PasteFull struct {
	ScrapeURL string `json:"scrape_url"`
	FullURL   string `json:"full_url"`
	Date      string `json:"date"`
	Key       string `json:"key"`
	Size      string `json:"size"`
	Expire    string `json:"expire"`
	Title     string `json:"title"`
	Syntax    string `json:"syntax"`
	User      string `json:"user"`
	Data      string `json:"data"`
	RFC3339   string `json:"time"`
}

// Meta Information: https://pastebin.com/api_scraping.php
// Content: http://pastebin.com/api_scrape_item.php

// QueryPastes returns metadata for the last 100 public pastes.
func QueryPastes(server string) ([]PasteMeta, error) {
	req, err := http.NewRequest("GET",
		fmt.Sprintf("https://%s/api_scraping.php?limit=100", server), nil)

	if err != nil {
		log.Fatal("Could not build http request", err)
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Could not do request due to %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Could not fetch response due to %v", err)
		return nil, err
	}

	var pastes []PasteMeta
	if err := json.Unmarshal(body, &pastes); err != nil {
		log.Printf("Could not decode response due to %v, body: %s", err, string(body))
		return nil, err
	}

	return pastes, err
}

// buffered channel where we store fetched PasteMeta objects
var chanPasteMeta = make(chan PasteMeta, 0x100)

// Worker is a generic set of fields to support graceful start/stop of a
// concurrent service.
type Worker struct {
	StopChan    chan bool
	StoppedChan chan bool
	Running     bool
}

// QueryWorker is a Worker that fetches paste metadata.
type QueryWorker struct {
	Worker
	SleepTime time.Duration
	APIServer string
	OutChan   chan PasteMeta
}

// Run starts a QueryWorker instance
func (w *QueryWorker) Run(wg *sync.WaitGroup) {
	if !w.Running {
		w.StopChan = make(chan bool)
		wg.Add(1)
		go w.doRun()
		w.Running = true
	}
}

// Stop stops a QueryWorker instance, blocks until finished
func (w *QueryWorker) Stop(wg *sync.WaitGroup) {
	if w.Running {
		w.StoppedChan = make(chan bool)
		close(w.StopChan)
		<-w.StoppedChan
		wg.Done()
		w.Running = false
	}
}

func (w *QueryWorker) doRun() {
	for {
		select {
		case <-w.StopChan:
			close(w.StoppedChan)
			return
		default:
			pastes, err := QueryPastes(w.APIServer)
			if err != nil && err != io.EOF {
				log.Printf("Could not query API for new pastes due to %v", err)
				// could not fetch paste, wait 10 seconds
				time.Sleep(1000 * time.Millisecond)
				continue
			}
			for _, paste := range pastes {
				select {
				case <-w.StopChan:
					close(w.StoppedChan)
					return
				default:
					chanPasteMeta <- paste
				}
			}
			// fetch every 30 seconds the most recent 100 pastes
			time.Sleep(w.SleepTime)
		}
	}
}

// FetchPaste fetches paste contents via the web API.
func FetchPaste(paste PasteMeta) (string, error) {
	url := paste.ScrapeURL
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Could build request %v due to %v", req, err)
		return "", err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Could not do request %v due to %v", req, err)
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Could not read response body %v due to %v", resp.Body, err)
		return "", err
	}

	return string(body), nil
}

// StorageModule represents a storage abstraction to handle pastes.
type StorageModule interface {
	StorePaste(PasteFull)
	Initialize() error
	Destroy() error
	Check(PasteMeta) bool
}

// FetchWorker is a Worker that fetches paste contents and stores them via
// storage modules.
type FetchWorker struct {
	Worker
	OutModules             []StorageModule
	PastebinURLReplacement string
}

// Run starts a FetchWorker instance
func (w *FetchWorker) Run(wg *sync.WaitGroup) {
	if !w.Running {
		w.StopChan = make(chan bool)
		wg.Add(1)
		go w.doFetch()
		w.Running = true
	}
}

// Stop stops a FetchWorker instance, blocks until finished
func (w *FetchWorker) Stop(wg *sync.WaitGroup) {
	if w.Running {
		w.StoppedChan = make(chan bool)
		close(w.StopChan)
		<-w.StoppedChan
		wg.Done()
		w.Running = false
	}
}

func (w *FetchWorker) doFetch() {
	// cleanup on shutdown
	defer func() {
		for _, s := range w.OutModules {
			err := s.Destroy()
			if err != nil {
				log.Fatal("Could not destroy storage module")
			}
		}
	}()

	// initialize storage modules
	for _, s := range w.OutModules {
		err := s.Initialize()
		if err != nil {
			log.Fatal("Could not initialize storage module", err)
		}
	}

	for {
		select {
		case <-w.StopChan:
			close(w.StoppedChan)
			return
		case paste := <-chanPasteMeta:
			check := true
			for _, s := range w.OutModules {
				if s.Check(paste) {
					check = false
					break
				}
			}
			if check {
				continue
			}

			if w.PastebinURLReplacement != "" {
				paste.ScrapeURL = strings.Replace(paste.ScrapeURL, "pastebin.com",
					w.PastebinURLReplacement, -1)
			}

			pasteData, err := FetchPaste(paste)
			if err != nil {
				log.Printf("Could not fetch paste due to %v", err)
				continue
			}

			i, err := strconv.ParseInt(paste.Date, 10, 64)
			if err != nil {
				log.Printf("Could not convert timestamp %v", paste.Date)
				continue
			}
			t := time.Unix(i, 0)
			rfc3339Time := t.Format(time.RFC3339)
			pasteFull := PasteFull{
				ScrapeURL: paste.ScrapeURL,
				FullURL:   paste.FullURL,
				Date:      paste.Date,
				Key:       paste.Key,
				Size:      paste.Size,
				Expire:    paste.Expire,
				Title:     paste.Title,
				Syntax:    paste.Syntax,
				User:      paste.User,
				Data:      pasteData,
				RFC3339:   rfc3339Time,
			}

			for _, s := range w.OutModules {
				go s.StorePaste(pasteFull)
			}
		}
	}
}
