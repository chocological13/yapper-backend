package util

import (
	"errors"
	"github.com/jackc/pgx/v5/pgtype"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var (
	ErrInvalidUUID = errors.New("Invalid UUID format")
)

// ParseUUIDParam extracts and parses a UUID parameter from the URL
func ParseUUIDParam(r *http.Request, prefix string) (pgtype.UUID, error) {
	path := strings.TrimPrefix(r.URL.Path, prefix)

	uuidStr := strings.Split(path, "/")[0]

	var uuid pgtype.UUID
	err := uuid.Scan(uuidStr)
	if err != nil {
		return pgtype.UUID{}, ErrInvalidUUID
	}

	return uuid, nil
}

// ReadString returns a string value from the query string, or the provided
// default value if no matching key could be found.
func ReadString(qs url.Values, key string, defaultValue string) string {
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}

	return s
}

// ReadCSV reads a string value from the query string and then splits it
// into a slice on the comma character.
func ReadCSV(qs url.Values, key string, defaultValues []string) []string {
	csv := qs.Get(key)

	if csv == "" {
		return defaultValues
	}

	return strings.Split(csv, ",")
}

// ReadInt reads a string value from the query string and converts it to an
// integer before returning.
// If the value couldn't be converted to an integer, then we record an
// error message in the provided Validator instance.
func ReadInt(qs url.Values, key string, defaultValue int, v *Validator) int {
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}

	return i
}
