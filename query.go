package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/elastic/go-elasticsearch/v8"
)

func RunQuery(es *elasticsearch.Client, query Query) (interface{}, error) {
	reqBody, _ := json.Marshal(query.Query)

	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithBodyString(string(reqBody)),
		es.Search.WithTrackTotalHits(true),
		es.Search.WithPretty(),
	)
	if err != nil {
		log.Printf("Greška pri izvršavanju upita %s: %v\n", query.Name, err)
		return nil, err
	}
	defer res.Body.Close()

	var r map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		log.Printf("Greška pri dekodiranju rezultata upita %s: %v\n", query.Name, err)
		return nil, err
	}

	return r, nil
}
