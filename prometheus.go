package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

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

func init() {
	prometheus.MustRegister(elasticQueryHits)
	prometheus.MustRegister(elasticQueryDurationMilliseconds)
}

func ExposeMetrics(port int) {
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
