package shortener

// Encoder translates between integer IDs and short code strings.
type Encoder interface {
	// Encode converts an integer ID to a short code string.
	Encode(id int64) (string, error)
	// Decode converts a short code string back to its integer ID.
	Decode(code string) (int64, error)
}
