package shortener

// Encoder translates integer IDs to short code strings.
type Encoder interface {
	Encode(id int64) (string, error)
}
