package util

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Envelope map[string]any

func ReadJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := decodeJSON(dec, dst); err != nil {
		return err
	}

	return nil
}

func ReadJSONForm(jsonValue string, dst any) error {
	dec := json.NewDecoder(strings.NewReader(jsonValue))
	dec.DisallowUnknownFields()

	if err := decodeJSON(dec, dst); err != nil {
		return err
	}

	return nil
}

func decodeJSON(dec *json.Decoder, dst any) error {
	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.UnmarshalTypeError
		var unmarshallTypeError *json.UnmarshalTypeError
		var invalidMarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("Body contains badly-formed JSON at character %d", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("Body contains badly-formed JSON")

		case errors.As(err, &unmarshallTypeError):
			if unmarshallTypeError.Field != "" {
				return fmt.Errorf("Body contains incorrect JSON type for field %q", unmarshallTypeError.Field)
			}
			return fmt.Errorf("Body contains incorrect JSON type (at character %d", unmarshallTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("Body must not be empty")

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field")
			return fmt.Errorf("Body contains unknown key %s", fieldName)

		case errors.As(err, &maxBytesError):
			return fmt.Errorf("Body must not be larger than %d bytes", maxBytesError.Limit)

		case errors.As(err, &invalidMarshalError):
			panic(err)

		default:
			return err
		}
	}

	return nil
}

func WriteJSON(w http.ResponseWriter, status int, data Envelope, headers http.Header) error {
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}

	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}
