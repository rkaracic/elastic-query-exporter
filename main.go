package main

import (
    "context"
    "crypto/tls"
    "crypto/x509"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "os"
    "time"

    "github.com/elastic/go-elasticsearch/v8"
)

func main() {
    debug := flag.Bool("debug", false, "Pokreni u debug modu")
    flag.Parse()

    configPath := "./config.json"
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
        Logger: &elasticsearch.DefaultLogger{},
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

    ExposeMetrics(config.PrometheusPort)
}
