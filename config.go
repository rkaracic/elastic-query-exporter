package main

import (
	"encoding/json"
	"io/ioutil"
)

type Config struct {
	ElasticsearchURL      string   `json:"elasticsearch_url"`
	ElasticsearchUsername string   `json:"elasticsearch_username"`
	ElasticsearchPassword string   `json:"elasticsearch_password"`
	ElasticsearchCACertPath string `json:"elasticsearch_ca_cert_path"`
	QueriesPath           string   `json:"queries_path"`
	PrometheusPort        int      `json:"prometheus_port"`
	Queries               []Query  `json:"queries"`
}

type Query struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"` // raw ili default
	Query        map[string]interface{} `json:"query"`
	Aggregations []Aggregation          `json:"aggregations"`
}

type Aggregation struct {
	Name string `json:"name"`
	Type string `json:"type"` // min, max, sum, itd.
}

func LoadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
