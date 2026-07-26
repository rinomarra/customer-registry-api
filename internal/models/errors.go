package models

import (
	"sort"
	"strings"
)

// ValidationErrors associa ogni campo non valido a un messaggio verificabile.
type ValidationErrors map[string]string

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}

	keys := make([]string, 0, len(e))
	for key := range e {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+": "+e[key])
	}
	return strings.Join(parts, "; ")
}

func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}
