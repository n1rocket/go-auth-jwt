package request

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Decoder provides a generic way to decode and validate HTTP requests
type Decoder struct {
	trimStrings bool
}

// NewDecoder creates a new request decoder
func NewDecoder() *Decoder {
	return &Decoder{
		trimStrings: true,
	}
}

// DecodeAndValidate decodes JSON request body and validates it
func (d *Decoder) DecodeAndValidate(r *http.Request, v Validator) error {
	if err := d.Decode(r, v); err != nil {
		return err
	}

	if d.trimStrings {
		if trimmer, ok := v.(StringTrimmer); ok {
			trimmer.TrimStrings()
		}
	}

	return v.Validate()
}

// Decode decodes JSON request body into the given struct
func (d *Decoder) Decode(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("request body is empty")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(v); err != nil {
		if strings.Contains(err.Error(), "unknown field") {
			return fmt.Errorf("request contains unknown field")
		}
		return fmt.Errorf("invalid JSON format: %w", err)
	}

	return nil
}

// Validator interface for request validation
type Validator interface {
	Validate() error
}

// StringTrimmer interface for trimming string fields
type StringTrimmer interface {
	TrimStrings()
}
