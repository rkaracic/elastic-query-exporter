package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
    "github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Struktura za konfiguraciju
type Config struct {
	ElasticsearchURL      string   `json:"elasticsearch_url"`
	ElasticsearchUsername string   `json:"elasticsearch_username"`
	ElasticsearchPassword string   `json:"elasticsearch_password"`
	ElasticsearchCACertPath string `json:"elasticsearch_ca_cert_path"`
	QueriesPath           string   `json:"queries_path"`
	PrometheusPort        int      `json:"prometheus_port"`
	Queries               []Query  `json:"queries"`
}

// Struktura za upit
type Query struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"` // raw ili default
	Query        map[string]interface{} `json:"query"`
}

// Funkcija za učitavanje konfiguracije
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

// Funkcija za pokretanje upita
func RunQuery(es *elasticsearch.Client, query Query) (interface{}, error) {
	reqBody, _ := json.Marshal(query.Query)

	req := esapi.SearchRequest{
		Body: ioutil.NopCloser(bytes.NewReader(reqBody)),
	}

	res, err := req.Do(context.Background(), es)
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

// Funkcija za izlaganje metrika
func ExposeMetrics(port int) {
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

// Prometrike za upite
var (
	elasticQueryHits = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "elastic_query_hits",
			Help: "Broj zapisa koji zadovoljavaju upit",
		},
		[]string{"query_name"},
	)

	elasticQueryDurationMilliseconds = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "elastic_query_duration_milliseconds",
			Help: "Vrijeme izvršavanja upita u milisekundama",
		},
		[]string{"query_name"},
	)
)

func main() {
	debug := os.Getenv("DEBUG") == "true"
	if debug {
		log.Println("Pokrenuto u debug modu")
	}

    configPath := "/app/config/config.json"
    config, err := LoadConfig(configPath)
    if err != nil {
        log.Fatal(err)
    }

	caCert, err := ioutil.ReadFile(config.ElasticsearchCACertPath)
	if err != nil {
		log.Fatal(err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}

	esConfig := elasticsearch.Config{
		Addresses: []string{config.ElasticsearchURL},
		Username:  config.ElasticsearchUsername,
		Password:  config.ElasticsearchPassword,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	es, err := elasticsearch.NewClient(esConfig)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			for _, query := range config.Queries {
				start := time.Now()
				result, err := RunQuery(es, query)
				if err != nil {
					log.Printf("Greška pri izvršavanju upita %s: %v\n", query.Name, err)
					continue
				}

				hits := result.(map[string]interface{})["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64)
				elasticQueryHits.WithLabelValues(query.Name).Set(hits)

				duration := time.Since(start).Milliseconds()
				elasticQueryDurationMilliseconds.WithLabelValues(query.Name).Set(float64(duration))

				log.Printf("Upit %s izvršen za %v ms\n", query.Name, duration)
			}
			time.Sleep(10 * time.Second)
		}
	}()

	prometheus.MustRegister(elasticQueryHits)
	prometheus.MustRegister(elasticQueryDurationMilliseconds)

	ExposeMetrics(config.PrometheusPort)
}
