# Dominio applicativo

## Cliente

Soggetto aziendale identificato dalla partita IVA.

1. Ragione sociale obbligatoria, massimo 200 caratteri.
2. Partita IVA obbligatoria, 11 cifre, univoca nell'archivio.
3. Email obbligatoria, indirizzo semplice valido.
4. Telefono, indirizzo e note facoltativi.
5. Stato solo `attivo` o `sospeso`; in creazione il default è `attivo`.
6. In `PUT` lo stato è obbligatorio: l'operazione sostituisce tutti i dati modificabili.

Lo stato sospeso è una classificazione amministrativa, non un blocco tecnico.

## Referente

Persona associata a un solo cliente.

1. Nome, cognome ed email obbligatori; ruolo facoltativo.
2. La stessa email può comparire su referenti diversi: non identifica univocamente una persona.
3. Si indirizza sempre tramite il cliente proprietario.
4. Non è possibile crearlo per un cliente inesistente.

## Relazione e cancellazione

Relazione uno-a-molti. La chiave esterna `contacts.client_id` usa `ON DELETE CASCADE`: eliminare un cliente elimina tutti i suoi referenti nella stessa operazione. Nessun referente orfano, nessun recupero, nessun soft delete. La cancellazione è riservata agli amministratori.

## Identità e autorizzazione

- `amministratore`: legge, crea, modifica, cancella.
- `operatore`: legge e crea; modifica e cancellazione restituiscono `403`.

Il login genera un token opaco casuale: nel database è salvato solo l'hash SHA-256, il valore in chiaro è restituito una sola volta. Password con PBKDF2-HMAC-SHA-256, salt casuale e confronto constant-time. Scadenza configurabile, revoca con il logout.

## Conflitti e consistenza

L'unicità della partita IVA è protetta dalla validazione applicativa e da un vincolo `UNIQUE`. In concorrenza l'invariante è garantita dal database e l'API traduce la violazione in `409 vat_number_conflict`. Tutte le date sono in UTC, formato RFC 3339.
