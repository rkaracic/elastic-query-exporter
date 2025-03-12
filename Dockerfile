FROM ubuntu:24.04

RUN apt-get update && apt-get install -y ca-certificates openssl
RUN update-ca-certificates

WORKDIR /app

COPY go.mod go.sum ./

RUN apt update && apt install -y golang-go
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-extldflags "-static"' -o elastic_query_exporter main.go

CMD ["./elastic_query_exporter", "-debug"]

EXPOSE 8000