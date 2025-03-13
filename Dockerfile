FROM golang:1.22 AS builder

WORKDIR /app

# Kopirajte Go module datoteke
COPY go.mod go.sum ./

# Preuzmite ovisnosti
RUN go mod download

# Kopirajte ostatak koda
COPY . .

# Kompajlirajte aplikaciju
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-extldflags "-static"' -o elastic_query_exporter main.go

# Finalni image
FROM alpine:latest

WORKDIR /app

# Kopirajte binarni file iz buildera
COPY --from=builder /app/elastic_query_exporter .

# Kopirajte konfiguracijsku datoteku (ako nije mountana)
COPY config.json ./config.json

# Postavite defaultnu komandu
CMD ["./elastic_query_exporter"]