package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	v7 "github.com/elastic/go-elasticsearch/v7"
	v8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Struktura za konfiguraciju
type Config struct {
	ElasticsearchURL        string  `json:"elasticsearch_url"`
	ElasticsearchUsername   string  `json:"elasticsearch_username"`
	ElasticsearchPassword   string  `json:"elasticsearch_password"`
	ElasticsearchCACertPath string  `json:"elasticsearch_ca_cert_path"`
	ElasticsearchVersion    int     `json:"elasticsearch_version"`
	InsecureSkipVerify      bool    `json:"insecure_skip_verify"`
	QueriesPath             string  `json:"queries_path"`
	PrometheusPort          int     `json:"prometheus_port"`
	QueryInterval           int     `json:"query_interval"` // Globalni interval
	Queries                 []Query `json:"queries"`
}

// Struktura za upit
type Query struct {
	Name       string         `json:"name"`
	Type       string         `json:"type"`       // raw ili default
	QueryFile  string         `json:"query_file"` // Putanja do datoteke s upitom
	MetricName string         `json:"metric_name"`
	Labels     []LabelMapping `json:"labels"`
	ValuePath  string         `json:"value_path"`
	Interval   *int           `json:"interval"` // Sekunde
}

type LabelMapping struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// Mapa za držanje registriranih metrika
var registeredMetrics = make(map[string]*prometheus.GaugeVec)

// Funkcija za učitavanje konfiguracije
func LoadConfig(path string) (*Config, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("Greška pri učitavanju konfiguracije: %v\n", err)
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Printf("Greška pri dekodiranju konfiguracije: %v\n", err)
		return nil, err
	}

	return &config, nil
}

// Funkcija za učitavanje querya iz query filea
func LoadQueryFromFile(path string) (map[string]interface{}, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("greška pri učitavanju upita iz datoteke %s: %v", path, err)
	}

	var query map[string]interface{}
	err = json.Unmarshal(data, &query)
	if err != nil {
		return nil, fmt.Errorf("greška pri dekodiranju upita iz datoteke %s: %v", path, err)
	}

	return query, nil
}

// Funkcija za pokretanje upita
func RunQuery(es interface{}, query Query) (interface{}, error) {
	log.Printf("Pokrećem upit: %s\n", query.Name)

	queryData, err := LoadQueryFromFile(query.QueryFile)
	if err != nil {
		log.Printf("Greška pri učitavanju upita %s: %v\n", query.Name, err)
		return nil, err
	}

	reqBody, _ := json.Marshal(queryData)

	var res io.ReadCloser

	if es7, ok := es.(*v7.Client); ok {
		res7, err := es7.Search(
			es7.Search.WithContext(context.Background()),
			es7.Search.WithBody(bytes.NewReader(reqBody)),
			es7.Search.WithTrackTotalHits(true),
		)
		if err != nil {
			log.Printf("Greška pri izvršavanju upita %s: %v\n", query.Name, err)
			return nil, err
		}
		defer res7.Body.Close()
		res = res7.Body
	} else if es8, ok := es.(*v8.Client); ok {
		res8, err := es8.Search(
			es8.Search.WithContext(context.Background()),
			es8.Search.WithBody(bytes.NewReader(reqBody)),
			es8.Search.WithTrackTotalHits(true),
		)
		if err != nil {
			log.Printf("Greška pri izvršavanju upita %s: %v\n", query.Name, err)
			return nil, err
		}
		defer res8.Body.Close()
		res = res8.Body
	} else {
		return nil, fmt.Errorf("nepoznati klijent")
	}

	// Dodajte zapisivanje sirovog odgovora u log
	rawResponse, err := ioutil.ReadAll(res)
	if err != nil {
		log.Printf("Greška pri čitanju odgovora za upit %s: %v\n", query.Name, err)
		return nil, err
	}
	log.Printf("Odgovor Elasticsearch-a za upit %s: %s\n", query.Name, string(rawResponse))

	// Dekodiranje JSON odgovora iz već pročitanog rawResponse
	var r map[string]interface{}
	err = json.Unmarshal(rawResponse, &r)
	if err != nil {
		log.Printf("Greška pri dekodiranju rezultata upita %s: %v\n", query.Name, err)
		return nil, err
	}

	log.Printf("Rezultat upita %s dekodiran uspješno\n", query.Name)

	return r, nil
}

// Funkcija za obradu rezultata
func processResult(result interface{}, query Query) {
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		log.Printf("Rezultat za upit %s nije validan: %v\n", query.Name, result)
		return
	}

	aggregationsMap, ok := resultMap["aggregations"].(map[string]interface{})
	if !ok {
		log.Printf("Rezultat za upit %s nema 'aggregations': %v\n", query.Name, resultMap)
		return
	}

	buckets, ok := aggregationsMap["0"].(map[string]interface{})["buckets"].([]interface{})
	if !ok {
		log.Printf("Rezultat za upit %s nema 'buckets': %v\n", query.Name, aggregationsMap["0"])
		return
	}

	metric, ok := registeredMetrics[query.MetricName]
	if !ok {
		metric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: query.MetricName,
				Help: fmt.Sprintf("Metrika za upit %s", query.Name),
			},
			append([]string{"query_name"}, getLabelNames(query.Labels)...),
		)
		prometheus.MustRegister(metric)
		registeredMetrics[query.MetricName] = metric
	}

	for _, bucket := range buckets {
		bucketMap, ok := bucket.(map[string]interface{})
		if !ok {
			log.Printf("Bucket nije u očekivanom formatu: %v\n", bucket)
			continue
		}

		value, err := getPathValue(bucketMap, query.ValuePath)
		if err != nil {
			log.Printf("Greška pri izvlačenju vrijednosti iz bucketa: %v\n", err)
			continue
		}

		labelValues := []string{query.Name}
		for _, label := range query.Labels {
			labelValue, err := getPathValue(bucketMap, label.Path)
			if err != nil {
				log.Printf("Greška pri izvlačenju labela iz bucketa: %v\n", err)
				continue
			}
			labelValues = append(labelValues, fmt.Sprintf("%v", labelValue))
		}

		metric.WithLabelValues(labelValues...).Set(value.(float64))
	}
}

func getPathValue(data map[string]interface{}, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	current := data
	for _, part := range parts {
		if current == nil {
			return nil, fmt.Errorf("putanja %s nije validna", path)
		}
		value, ok := current[part]
		if !ok {
			return nil, fmt.Errorf("putanja %s nije validna", path)
		}
		if mapValue, isMap := value.(map[string]interface{}); isMap {
			current = mapValue
		} else {
			return value, nil
		}
	}
	return current, nil
}

func getLabelNames(labels []LabelMapping) []string {
	names := []string{}
	for _, label := range labels {
		names = append(names, label.Name)
	}
	return names
}

// Funkcija za izlaganje metrika
func ExposeMetrics(port int) {
	log.Println("Pokrećem izlaganje metrika...")
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

	var tlsConfig *tls.Config
	if config.ElasticsearchCACertPath != "" {
		caCert, err := ioutil.ReadFile(config.ElasticsearchCACertPath)
		if err != nil {
			log.Fatal(err)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		tlsConfig = &tls.Config{
			RootCAs: caCertPool,
		}
	} else {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify,
		}
	}

	var es interface{}
	if config.ElasticsearchVersion == 7 {
		esConfig := v7.Config{
			Addresses: []string{config.ElasticsearchURL},
			Username:  config.ElasticsearchUsername,
			Password:  config.ElasticsearchPassword,
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		}
		es7, err := v7.NewClient(esConfig)
		if err != nil {
			log.Fatal(err)
		}
		es = es7
	} else {
		esConfig := v8.Config{
			Addresses: []string{config.ElasticsearchURL},
			Username:  config.ElasticsearchUsername,
			Password:  config.ElasticsearchPassword,
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		}
		es8, err := v8.NewClient(esConfig)
		if err != nil {
			log.Fatal(err)
		}
		es = es8
	}

	go func() {
		for _, query := range config.Queries {
			interval := time.Duration(config.QueryInterval) * time.Second
			if query.Interval != nil {
				interval = time.Duration(*query.Interval) * time.Second
			}

			go func(q Query, i time.Duration) {
				for {
					start := time.Now()
					result, err := RunQuery(es, q)
					if err != nil {
						log.Printf("Greška pri izvršavanju upita %s: %v\n", q.Name, err)
						continue
					}

					log.Printf("Upit %s izvršen za %v ms\n", q.Name, time.Since(start).Milliseconds())
					processResult(result, q)

					duration := time.Since(start).Milliseconds()
					elasticQueryDurationMilliseconds.WithLabelValues(q.Name).Set(float64(duration))

					log.Printf("Metrike za upit %s ažurirane\n", q.Name)
					time.Sleep(i)
				}
			}(query, interval)
		}
	}()

	prometheus.MustRegister(elasticQueryHits)
	prometheus.MustRegister(elasticQueryDurationMilliseconds)

	log.Println("Exporter pokrenut.")

	ExposeMetrics(config.PrometheusPort)
}
