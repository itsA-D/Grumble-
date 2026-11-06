package pain

// Count is one labelled count in a browse list. Key is the stable slug or ISO
// country code; Count is always positive because callers hide empty entries.
type Count struct {
	Key   string
	Count int
}
