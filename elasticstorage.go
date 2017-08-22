package main

import (
	"context"
	elastic "gopkg.in/olivere/elastic.v5"
	"log"
)

type ElasticStorageModule struct {
	index      string
	elasticURL string
	client     *elastic.Client
}

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
	_ = searchResult

	if searchResult.TotalHits() > 0 {
		return false
	}

	return true
}

func (e *ElasticStorageModule) Destroy() error {
	_, err := e.client.Flush().Index(e.index).Do(context.Background())
	if err != nil {
		return err
	}
	return nil
}
