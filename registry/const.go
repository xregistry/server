package registry

var MAX_VARCHAR = 4096
var MAX_PROPNAME = 255

// Entity "add" options
type AddType int

const (
	ADD_ADD AddType = iota + 1
	ADD_UPDATE
	ADD_UPSERT
	ADD_PATCH // includes UPSERT
)

// COLLATE clause for case-insensitive string comparisons per spec
const FILTER_CI_COLLATE = "COLLATE utf8mb4_0900_ai_ci"

const HTML_EXP = "&#9662;" // Expanded json symbol for HTML output
const HTML_MIN = "&#9656;" // Minimized json symbol for HTML output
