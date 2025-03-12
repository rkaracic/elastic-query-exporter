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

## Upiti

Definirajte upite u `config.json` pod `queries` sekcijom.