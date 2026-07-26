# Customer Registry API

Piccola API REST in Go per gestire clienti e relativi referenti. Include autenticazione con token persistenti, ruoli applicativi, validazione dei dati, paginazione, filtri e storage SQLite.

## Entità

### Cliente

| Campo JSON | Obbligo | Vincolo |
|---|---|---|
| `ragione_sociale` | Obbligatorio | Stringa non vuota, massimo 200 caratteri |
| `partita_iva` | Obbligatorio | Esattamente 11 cifre, univoca |
| `email` | Obbligatorio | Email semplice valida, massimo 254 caratteri |
| `telefono` | Facoltativo | Massimo 30 caratteri; cifre, spazi e `+ ( ) . / -` |
| `indirizzo` | Facoltativo | Massimo 500 caratteri |
| `note` | Facoltativo | Massimo 2000 caratteri |
| `stato` | Obbligatorio in modifica | `attivo` o `sospeso`; in creazione il default è `attivo` |

### Referente

| Campo JSON | Obbligo | Vincolo |
|---|---|---|
| `nome` | Obbligatorio | Stringa non vuota, massimo 100 caratteri |
| `cognome` | Obbligatorio | Stringa non vuota, massimo 100 caratteri |
| `email` | Obbligatorio | Email semplice valida, massimo 254 caratteri |
| `ruolo` | Facoltativo | Massimo 100 caratteri |

## Endpoint

Tutti gli endpoint, tranne login e health check, richiedono `Authorization: Bearer <token>`.

| Metodo | Percorso | Ruolo | Risposte principali |
|---|---|---|---|
| `GET` | `/healthz` | Pubblico | 200, 503 |
| `POST` | `/api/v1/auth/login` | Pubblico | 200, 400, 401, 500 |
| `POST` | `/api/v1/auth/logout` | Autenticato | 204, 401, 500 |
| `GET` | `/api/v1/clienti` | Entrambi | 200, 400, 401, 500 |
| `POST` | `/api/v1/clienti` | Entrambi | 201, 400, 401, 409, 500 |
| `GET` | `/api/v1/clienti/{id}` | Entrambi | 200, 400, 401, 404, 500 |
| `PUT` | `/api/v1/clienti/{id}` | Amministratore | 200, 400, 401, 403, 404, 409, 500 |
| `DELETE` | `/api/v1/clienti/{id}` | Amministratore | 204, 400, 401, 403, 404, 500 |
| `GET` | `/api/v1/clienti/{id}/referenti` | Entrambi | 200, 400, 401, 404, 500 |
| `POST` | `/api/v1/clienti/{id}/referenti` | Entrambi | 201, 400, 401, 404, 500 |
| `GET` | `/api/v1/clienti/{id}/referenti/{referenteId}` | Entrambi | 200, 400, 401, 404, 500 |
| `PUT` | `/api/v1/clienti/{id}/referenti/{referenteId}` | Amministratore | 200, 400, 401, 403, 404, 500 |
| `DELETE` | `/api/v1/clienti/{id}/referenti/{referenteId}` | Amministratore | 204, 400, 401, 403, 404, 500 |

## Regole di validazione

- Gli spazi esterni vengono rimossi; le email vengono normalizzate in minuscolo.
- La partita IVA deve corrispondere a `^[0-9]{11}$` e non può appartenere a due clienti.
- Le email devono essere indirizzi semplici, non valori nel formato `Nome <email@dominio>`.
- `PUT` sostituisce l'intera risorsa: i campi obbligatori devono essere sempre presenti.
- I campi JSON sconosciuti e i corpi con più oggetti sono rifiutati con `400`.
- Eliminando un cliente vengono eliminati anche tutti i suoi referenti.

## Ruoli

- **amministratore**: lettura, creazione, modifica e cancellazione.
- **operatore**: sola lettura e creazione; modifica e cancellazione restituiscono `403`.

## Avvio locale

Prerequisiti: Go 1.23 o successivo, `CGO_ENABLED=1` e un compilatore C, richiesto dal driver SQLite. Con `go run ./cmd/api` l'API ascolta su `http://localhost:8080` e crea il database in `./data/customer-registry.db`. Le credenziali iniziali e i parametri di esecuzione sono configurabili da variabili d'ambiente.

La specifica degli endpoint è in [`docs/api.md`](docs/api.md); le regole di dominio sono in [`docs/domain.md`](docs/domain.md).
