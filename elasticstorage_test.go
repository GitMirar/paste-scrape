package main

import (
	"testing"
)

func _TestElasticStorageModule(t *testing.T) {
	e := ElasticStorageModule{
		index:      "pastebin-test",
		elasticURL: "http://elk.lab",
	}

	err := e.Initialize()
	if err != nil {
		t.Fail()
	}

	/*
		err = e.StorePaste(paste)
		if err != nil {
			t.Fail()
		}
	*/

	err = e.Destroy()
	if err != nil {
		t.Fail()
	}
}
