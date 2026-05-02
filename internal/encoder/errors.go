// Package encoder provides short-code encoding and decoding for the shortener service.
package encoder

import "errors"

// ErrNegativeID is returned when Encode is called with a negative integer.
var ErrNegativeID = errors.New("id must be non-negative")

// ErrInvalidCode is returned when Decode is called with an unrecognizable short code.
var ErrInvalidCode = errors.New("invalid short code")
