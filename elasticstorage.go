package main

import (
	"context"
	"log"

	elastic "gopkg.in/olivere/elastic.v5"
)

const mapping = `
{
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
	client       *elastic.Client
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
	log.Printf("Elasticsearch returned with code %d and version %s", code, info.Version.Number)

	exists, err := e.client.IndexExists(e.index).Do(context.Background())
	if err != nil {
		return err
	}

	if !exists {
		log.Printf("Creating new index %s", e.index)

		e.client.CreateIndex(e.index).Do(context.Background())
	}

	return nil
}

// StorePaste stores a single paste in the storage backend.
func (e *ElasticStorageModule) StorePaste(paste PasteFull) {
	_, err := e.client.Index().
		Index(e.index).
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
