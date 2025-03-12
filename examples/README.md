# Primjeri konfiguracija

Ovdje se nalaze primjeri kako upisivati upite u `config.json` datoteku za različite scenarije:

## Primjeri

### `http_no_auth.json`

- **`elasticsearch_url`**: URL adresa Elasticsearch servera. U ovom slučaju, koristi se HTTP protokol na lokalnoj adresi (`http://localhost:9200`). Mogućnosti:
  - `http://localhost:9200` za HTTP povezivanje.
  - `https://localhost:9200` za HTTPS povezivanje.

- **`queries_path`**: Putanja do direktorija gdje će se nalaziti dodatni upiti ili konfiguracije. U ovom primjeru nije korištena. Mogućnosti:
  - Može biti prazna ili nepostojeća putanja ako se ne koriste dodatni upiti.

- **`prometheus_port`**: Port na kojem će exporter izlagati metrike za Prometheus. U ovom slučaju, koristi se port 8000. Mogućnosti:
  - Bilo koji slobodan port (npr. 8000, 8080, itd.).

- **`queries`**: Lista upita koji će se izvršavati na Elasticsearchu.
  - **`name`**: Naziv upita. Mogućnosti:
    - Bilo koji string koji opisuje upit (npr. "upit1", "broj_dokumenata", itd.).
  - **`type`**: Tip upita. Mogućnosti:
    - `"default"`: Upit će se izvršiti kao standardni Elasticsearch upit.
    - `"raw"`: Upit će se izvršiti kao raw JSON upit.
  - **`query`**: Sam upit koji će se izvršiti. U ovom slučaju, koristi se jednostavan match upit. Mogućnosti:
    - Bilo koji validan Elasticsearch upit (npr. match, term, bool, itd.).

### `https_with_auth.json`

- **`elasticsearch_username`**: Username za autentifikaciju na Elasticsearch serveru. Mogućnosti:
  - Vaš stvarni username za Elasticsearch.

- **`elasticsearch_password`**: Password za autentifikaciju na Elasticsearch serveru. Mogućnosti:
  - Vaš stvarni password za Elasticsearch.

Ostale postavke su iste kao u primjeru `http_no_auth.json`.

### `https_with_ca_cert.json`

- **`elasticsearch_ca_cert_path`**: Putanja do CA certifikata koji će se koristiti za provjeru identiteta Elasticsearch servera. Mogućnosti:
  - Putanja do vašeg CA certifikata (npr. `/path/to/ca.crt`).

Ostale postavke su iste kao u primjeru `http_no_auth.json`.

### `https_with_ca_cert_and_auth.json`

Ova konfiguracija kombinira sve prethodne postavke:
- Koristi HTTPS povezivanje.
- Uključuje autentifikaciju s username i password.
- Koristi CA certifikat za dodatnu sigurnost.

Sve postavke su iste kao u prethodnim primjerima.
