package models

import "testing"

func TestClientValidation(t *testing.T) {
	tests := []struct {
		name      string
		input     ClientInput
		wantField string
	}{
		{
			name: "partita IVA corta",
			input: ClientInput{
				BusinessName: "Acme S.r.l.",
				VATNumber:    "123",
				Email:        "info@acme.test",
				Status:       ClientStatusActive,
			},
			wantField: "partita_iva",
		},
		{
			name: "email non valida",
			input: ClientInput{
				BusinessName: "Acme S.r.l.",
				VATNumber:    "12345678901",
				Email:        "non-valida",
				Status:       ClientStatusActive,
			},
			wantField: "email",
		},
		{
			name: "stato non valido",
			input: ClientInput{
				BusinessName: "Acme S.r.l.",
				VATNumber:    "12345678901",
				Email:        "info@acme.test",
				Status:       "archiviato",
			},
			wantField: "stato",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			errs := test.input.ValidateForUpdate()
			if _, ok := errs[test.wantField]; !ok {
				t.Fatalf("atteso errore su %s, ottenuto %#v", test.wantField, errs)
			}
		})
	}
}

func TestClientCreateDefaultsStatusToActive(t *testing.T) {
	input := ClientInput{
		BusinessName: "Acme S.r.l.",
		VATNumber:    "12345678901",
		Email:        "info@acme.test",
	}
	if errs := input.ValidateForCreate(); errs.HasErrors() {
		t.Fatalf("input valido rifiutato: %#v", errs)
	}
}

func TestContactValidation(t *testing.T) {
	input := ContactInput{
		FirstName: "Mario",
		LastName:  "Rossi",
		Email:     "mario.rossi@example.test",
		Role:      "Responsabile acquisti",
	}
	if errs := input.Validate(); errs.HasErrors() {
		t.Fatalf("referente valido rifiutato: %#v", errs)
	}

	input.FirstName = ""
	if errs := input.Validate(); errs["nome"] == "" {
		t.Fatalf("atteso errore sul nome, ottenuto %#v", errs)
	}
}
