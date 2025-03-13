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
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Struktura za konfiguraciju
type Config struct {
	ElasticsearchURL        string  `json:"elasticsearch_url"`
	ElasticsearchUsername   string  `json:"elasticsearch_username"`
	ElasticsearchPassword   string  `json:"elasticsearch_password"`
	ElasticsearchCACertPath string  `json:"elasticsearch_ca_cert_path"`
	InsecureSkipVerify      bool    `json:"insecure_skip_verify"`
	QueriesPath             string  `json:"queries_path"`
	PrometheusPort          int     `json:"prometheus_port"`
	QueryInterval           int     `json:"query_interval"` // Globalni interval
	Queries                 []Query `json:"queries"`
}

// Struktura za upit
type Query struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"` // raw ili default
	Query      map[string]interface{} `json:"query"`
	MetricName string                 `json:"metric_name"`
	Labels     []LabelMapping         `json:"labels"`
	ValuePath  string                 `json:"value_path"`
	Interval   *int                   `json:"interval"` // Sekunde
}

type LabelMapping struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

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

// Funkcija za pokretanje upita
func RunQuery(es *elasticsearch.Client, query Query) (interface{}, error) {
	log.Printf("Pokrećem upit: %s\n", query.Name)

	reqBody, _ := json.Marshal(query.Query)

	res, err := es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithBody(bytes.NewReader(reqBody)),
		es.Search.WithTrackTotalHits(true),
	)
	if err != nil {
		log.Printf("Greška pri izvršavanju upita %s: %v\n", query.Name, err)
		return nil, err
	}
	defer res.Body.Close()

	log.Printf("Upit %s izvršen uspješno\n", query.Name)

	var r map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&r)
	if err != nil {
		log.Printf("Greška pri dekodiranju rezultata upita %s: %v\n", query.Name, err)
		return nil, err
	}

	log.Printf("Rezultat upita %s dekodiran uspješno\n", query.Name)
	log.Printf("Raw rezultat upita %s: %+v\n", query.Name, r)

	return r, nil
}

// Funkcija za obradu rezultata
func processResult(result interface{}, query Query) {
	// Provjerite da li je rezultat validan
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

	metricName := query.MetricName
	labels := query.Labels
	valuePath := query.ValuePath

	for _, bucket := range buckets {
		bucketMap, ok := bucket.(map[string]interface{})
		if !ok {
			log.Printf("Bucket nije u očekivanom formatu: %v\n", bucket)
			continue
		}

		value, err := getPathValue(bucketMap, valuePath)
		if err != nil {
			log.Printf("Greška pri izvlačenju vrijednosti iz bucketa: %v\n", err)
			continue
		}

		labelValues := []string{}
		for _, label := range labels {
			labelValue, err := getPathValue(bucketMap, label.Path)
			if err != nil {
				log.Printf("Greška pri izvlačenju labela iz bucketa: %v\n", err)
				continue
			}
			labelValues = append(labelValues, labelValue.(string))
		}

		// Postavite metriku ako je sve validno
		metric := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: metricName,
				Help: fmt.Sprintf("Metrika za upit %s", query.Name),
			},
			append([]string{"query_name"}, getLabelNames(labels)...),
		)
		prometheus.MustRegister(metric)
		metric.WithLabelValues(append([]string{query.Name}, labelValues...)...).Set(value.(float64))
	}
}

func getPathValue(data map[string]interface{}, path string) (interface{}, error) {
	parts := strings.Split(path, ".")
	current := data
	for _, part := range parts {
		if current == nil {
			return nil, fmt.Errorf("putanja %s nije validna", path)
		}
		current, ok := current[part]
		if !ok {
			return nil, fmt.Errorf("putanja %s nije validna", path)
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
		for _, query := range config.Queries {
			var interval time.Duration
			if query.Interval != nil {
				interval = time.Duration(int64(*query.Interval)) * time.Second
			} else {
				interval = time.Duration(config.QueryInterval) * time.Second
			}

			go func(q Query, i time.Duration) {
				for {
					// Izvršavanje upita...
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
