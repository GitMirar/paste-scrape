package main

import (
	"context"
	"fmt"
	"log"
	"time"

	elastic "gopkg.in/olivere/elastic.v5"
)

const mapping = `{
   "settings":{
      "number_of_shards":1,
      "number_of_replicas":0
   },
   "mappings":{
      "_default_":{
         "_all":{
            "enabled":true
         }
      },
      "paste":{
         "properties":{
            "key":{
               "type":"keyword"
            },
            "data":{
               "type":"text",
               "store":true,
               "fielddata":true
            },
            "size":{
               "type":"long"
            },
            "syntax":{
               "type":"keyword"
            },
            "date":{
               "type":"date",
               "format":"epoch_second"
            },
            "expire":{
               "type":"date",
               "format":"epoch_second"
            }
         }
      }
   }
}`

// ElasticStorageModule is a StorageModule that stores pastes in an
// Elasticsearch instance.
type ElasticStorageModule struct {
	index        string
	elasticURL   string
	dailyIndexes bool
	useIndex     string
	lastChk      time.Time
	client       *elastic.Client
}

func (e *ElasticStorageModule) makeIndexIfNotExists() error {
	index := e.index
	if e.dailyIndexes {
		index = time.Now().Format(fmt.Sprintf("%s-2006-1-2", e.index))
	}

	exists, err := e.client.IndexExists(index).Do(context.Background())
	if err != nil {
		return err
	}

	if !exists {
		log.Printf("Creating new index %s", index)
		e.client.CreateIndex(index).Body(mapping).Do(context.Background())
	}

	e.useIndex = index
	return nil
}

// Initialize prepares the storage modules for use.
func (e *ElasticStorageModule) Initialize() error {
	log.Printf("Connecting to %v", e.elasticURL)
	var err error
	e.client, err = elastic.NewSimpleClient(elastic.SetURL(e.elasticURL))
	if err != nil {
		return err
	}

	info, code, err := e.client.Ping(e.elasticURL).Do(context.Background())
	if err != nil {
		return err
	}
	log.Printf("Elasticsearch returned with code %d and version %s", code,
		info.Version.Number)

	err = e.makeIndexIfNotExists()
	if err != nil {
		return err
	}

	e.lastChk = time.Now()
	log.Printf("Using index %s", e.useIndex)
	return err
}

// StorePaste stores a single paste in the storage backend.
func (e *ElasticStorageModule) StorePaste(paste PasteFull) {
	if e.dailyIndexes && time.Since(e.lastChk) > 12*time.Hour {
		e.makeIndexIfNotExists()
		e.lastChk = time.Now()
	}
	_, err := e.client.Index().
		Index(e.useIndex).
		Type("paste").
		Id(paste.Key).
		BodyJson(paste).
		Do(context.Background())
	if err != nil {
		log.Printf("Could not store paste %s due to %v", paste.FullURL, err)
	}
}

// Check returns true if the given paste is new in the current storage backend,
// false otherwise.
func (e *ElasticStorageModule) Check(paste PasteMeta) bool {
	q := elastic.NewMatchQuery("_id", paste.Key)

	searchResult, err := e.client.Search().
		Index(e.index).
		Query(q).
		Pretty(true).
		Do(context.Background())
	if err != nil {
		log.Printf("Could not check paste due to %v", err)
		return true
	}

	return (searchResult.TotalHits() == 0)
}

// Destroy finishes all operations on the module.
func (e *ElasticStorageModule) Destroy() error {
	_, err := e.client.Flush().Index(e.index).Do(context.Background())
	return err
}
