package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

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

const api_url = "https://scrape.pastebin.com/api_scraping.php"

func QueryPastes() ([]PasteMeta, error) {
	url := api_url

	var postData = []byte(`{"limit": 100}`)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(postData))
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

	var pastes []PasteMeta
	if err := json.NewDecoder(resp.Body).Decode(&pastes); err != nil {
		log.Printf("Could not decode response due to %v", err)
		return nil, err
	}

	return pastes, err
}

// buffered channel where we store fetched PasteMeta objects
var chanPasteMeta = make(chan PasteMeta, 0x100)
var stopQueryWorker = false
var queryWorkerStopped = make(chan bool)

func QueryWorker() {
	for {
		pastes, err := QueryPastes()
		if err != nil {
			log.Printf("Could not query API for new pastes due to %v", err)
			// could not fetch paste, wait 10 seconds
			time.Sleep(10000 * time.Millisecond)
			continue
		}
		for _, paste := range pastes {
			select {
			case chanPasteMeta <- paste:
			default:
				log.Printf("Pipeling stalling, dropping pastes...")
			}
		}

		// fetch every 30 seconds the most recent 100 pastes
		time.Sleep(30000 * time.Millisecond)

		if stopQueryWorker {
			break
		}
	}

	queryWorkerStopped <- true
}

func FetchPaste(paste PasteMeta) (string, error) {
	url := paste.ScrapeURL
	req, err := http.NewRequest("GET", url, nil)
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

type StorageModule interface {
	StorePaste(paste PasteFull)
	Initialize() error
	Destroy() error
	Check(paste PasteMeta) bool
}

var stopFetchWorker = false
var fetchWorkerStopped = make(chan bool)

func FetchWorker(s StorageModule) {
	err := s.Initialize()
	if err != nil {
		log.Fatal("Could not initialize storage module", err)
	}

	for {
		if stopFetchWorker {
			break
		}
		time.Sleep(500 * time.Millisecond)

		var paste PasteMeta
		select {
		case paste = <-chanPasteMeta:
		default:
			continue
		}

		if !s.Check(paste) {
			continue
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
		go s.StorePaste(pasteFull)
	}

	err = s.Destroy()
	if err != nil {
		log.Fatal("Could not destroy storage module")
	}
	fetchWorkerStopped <- true
}
