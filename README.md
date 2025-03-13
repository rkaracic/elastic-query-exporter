# Elasticsearch Exporter

Elasticsearch Exporter je Go aplikacija dizajnirana za pokretanje upita na Elasticsearch i izlaganje rezultata kao metrike u Prometheus formatu. Projekt omogućava laku integraciju s postojećim monitoring sistemima i pruža detaljne informacije o performansama i stanju Elasticsearch baze.

## Sadržaj projekta

- **`main.go`**: Glavni program koji pokreće exporter.
- **`config.go`**: Konfiguracija za povezivanje s Elasticsearchom i definiranje upita.
- **`query.go`**: Funkcionalnost za pokretanje upita i obradu rezultata.
- **`prometheus.go`**: Funkcionalnost za izlaganje metrika u Prometheus formatu.
- **`go.mod` i `go.sum`**: Go module datoteke za upravljanje ovisnostima.
- **`examples/` direktorij**: Primjeri konfiguracija za različite scenarije povezivanja s Elasticsearchom.
- **`Dockerfile`**: Dockerfile za kreiranje kontejnera koji sadrži exporter.

## Pokretanje

1. **Instalirajte Go module**:
go mod tidy

2. **Pokrenite exporter**:
go run main.go -debug -log-file /path/to/log/exporter.log

3. **Kompajlirajte za drugu platformu**:
GOOS=linux GOARCH=amd64 go build -ldflags '-extldflags "-static"' -o elastic-query-exporter main.go

## Docker
1. **Kreirajte Docker sliku**:
docker build -t elastic-query-exporter .

2. **Pokrenite kontejner**:
docker run -d -p 8000:8000 elastic-query-exporter

3. **Pregledajte logove**:
docker logs -f <kontejner_id>

## Docker Compose
1. **Pokrenite kontejnere**:
docker-compose up -d

2. **Pregledajte logove**:
docker-compose logs -f


## Konfiguracija

Konfiguracija se nalazi u `config.json`. Postavite URL, username, password i CA certifikat za Elasticsearch.

### Tipovi upita

Postoje dva tipa upita koji se mogu koristiti: `default` i `raw`.

#### Default Type

Upiti tipa `default` su dizajnirani za jednostavno izlaganje broja zapisa koji zadovoljavaju upit. Ovi upiti se automatski ažuriraju i izlažaju kao metrike u Prometheus-u.

Primjer `default` upita:
{
    "name": "default_query",
    "type": "default",
    "query": {
        "bool": {
            "filter": [
                {
                    "match": {
                        "device_type": "stb"
                    }
                }
            ]
        }
    }
}

#### Raw Type

Upiti tipa `raw` omogućavaju izvršavanje kompleksnijih upita koji se ne moraju nužno ažurirati kao metrike. Ovi upiti se mogu koristiti za specifične slučajeve gdje je potrebno izvršiti složenije agregacije ili upite.

Primjer `raw` upita:
{
    "name": "raw_query",
    "type": "raw",
    "query": {
        "aggs": {
            "c858866f-a23d-44f0-8177-dadaa6d646f0": {
                "date_histogram": {
                    "field": "@timestamp",
                    "calendar_interval": "1d",
                    "time_zone": "Europe/Zagreb"
                }
            }
        },
        "size": 0,
        "query": {
            "bool": {
                "filter": [
                    {
                        "match_all": {}
                    },
                    {
                        "bool": {
                            "should": [
                                {
                                    "match": {
                                        "device_type": "stb"
                                    }
                                }
                            ],
                            "minimum_should_match": 1
                        }
                    },
                    {
                        "range": {
                            "@timestamp": {
                                "gte": "2025-03-05T06:44:53.137Z",
                                "lte": "2025-03-12T06:44:53.137Z",
                                "format": "strict_date_optional_time"
                            }
                        }
                    }
                ]
            }
        }
    }
}

## Upiti

Definirajte upite u `config.json` pod `queries` sekcijom.