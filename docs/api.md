# Specifica API

Base URL locale `http://localhost:8080`. Le richieste con corpo usano `Content-Type: application/json`.

## Formato degli errori

```json
{
  "error": {
    "code": "validation_failed",
    "message": "i dati del cliente non sono validi",
    "details": { "partita_iva": "deve contenere esattamente 11 cifre" }
  }
}
```

`details` è presente solo quando l'errore è associabile a campi specifici.

| HTTP | Codice JSON | Significato |
|---|---|---|
| 400 | `invalid_json` | Corpo assente, malformato o con campi sconosciuti |
| 400 | `validation_failed` | Valori non conformi ai vincoli |
| 401 | `unauthorized` / `invalid_credentials` | Token assente o non valido, credenziali errate |
| 403 | `forbidden` | Ruolo senza permesso |
| 404 | `client_not_found`, `contact_not_found`, `route_not_found` | Risorsa inesistente |
| 409 | `vat_number_conflict` | Partita IVA già usata |
| 500 | `internal_error` | Errore inatteso |

## Autenticazione

- **POST `/api/v1/auth/login`** — pubblico. Corpo `email` e `password`; `200` con token opaco, scadenza e utente. Credenziali errate: `401`.
- **POST `/api/v1/auth/logout`** — `204`, revoca il token corrente.
- **GET `/healthz`** — `200 {"status":"ok"}`, oppure `503 database_unavailable`.

## Clienti

Oggetto restituito: `id`, `ragione_sociale`, `partita_iva`, `email`, `telefono`, `indirizzo`, `note`, `stato`, `creato_il`, `aggiornato_il`.

**GET `/api/v1/clienti`** — entrambi i ruoli. Filtri: `pagina` (default 1), `elementi_per_pagina` (default 20, da 1 a 100), `q` (parziale su ragione sociale, email, partita IVA), `partita_iva` (11 cifre, esatto), `stato` (`attivo`/`sospeso`). Risposta con `data` e `paginazione`. Filtro non valido: `400`.

**POST `/api/v1/clienti`** — entrambi i ruoli. Obbligatori `ragione_sociale`, `partita_iva`, `email`. `201` con header `Location`. Validazione fallita: `400` con `details` per campo. Partita IVA duplicata: `409`.

**GET `/api/v1/clienti/{id}`** — entrambi i ruoli. `404 client_not_found` se assente; `400` se l'identificativo non è numerico.

**PUT `/api/v1/clienti/{id}`** — solo amministratore. Sostituisce tutti i campi modificabili: obbligatori presenti, incluso `stato`. Operatore: `403`.

**DELETE `/api/v1/clienti/{id}`** — solo amministratore. `204`; elimina anche i referenti collegati. Operatore: `403`.

## Referenti

Oggetto restituito: `id`, `cliente_id`, `nome`, `cognome`, `email`, `ruolo`, `creato_il`, `aggiornato_il`.

**GET `/api/v1/clienti/{id}/referenti`** — entrambi i ruoli. Filtri `pagina`, `elementi_per_pagina`, `q` (parziale su nome, cognome, email, ruolo). Cliente inesistente: `404 client_not_found`.

**POST `/api/v1/clienti/{id}/referenti`** — entrambi i ruoli. Obbligatori `nome`, `cognome`, `email`; `ruolo` facoltativo. `201` con `Location`.

**GET `/api/v1/clienti/{id}/referenti/{referenteId}`** — entrambi i ruoli. Se il referente non esiste **o non appartiene al cliente indicato**: `404 contact_not_found`.

**PUT `/api/v1/clienti/{id}/referenti/{referenteId}`** — solo amministratore. Sostituzione completa. Operatore: `403`.

**DELETE `/api/v1/clienti/{id}/referenti/{referenteId}`** — solo amministratore. `204`. Operatore: `403`.
