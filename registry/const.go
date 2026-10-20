package registry

// Entity "add" options
type AddType int

const (
	ADD_ADD AddType = iota + 1
	ADD_UPDATE
	ADD_UPSERT
	ADD_PATCH // includes UPSERT
)

const HTML_EXP = "&#9662;" // Expanded json symbol for HTML output
const HTML_MIN = "&#9656;" // Minimized json symbol for HTML output
