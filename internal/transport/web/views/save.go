package views

// SaveButton is the server-rendered saved toggle. Saved is authoritative on
// first paint; HTMX only replaces this fragment.
type SaveButton struct {
	PainID    string
	Saved     bool
	CSRFToken string
	// ReturnTo is sent for non-JS POST/DELETE redirects.
	ReturnTo string
	// FromSaved makes a removal replace the whole row and update the page OOB.
	FromSaved bool
	// Error is an inline message, currently the saved-list cap.
	Error string
}
