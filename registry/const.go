package registry

var MAX_VARCHAR = 4096
var MAX_PROPNAME = 255

const DOCVIEW_BASE = "#"

// Entity "add" options
type AddType int

const (
	ADD_ADD AddType = iota + 1
	ADD_UPDATE
	ADD_UPSERT
	ADD_PATCH // includes UPSERT
)

const (
	FILTER_PRESENT = iota + 1
	FILTER_ABSENT
	FILTER_EQUAL
	FILTER_NOT_EQUAL
)

const HTML_EXP = "&#9662;" // Expanded json symbol for HTML output
const HTML_MIN = "&#9656;" // Minimized json symbol for HTML output

const ANCESTOR_TBD = "$TBD"

// For entity.AccessMode
const (
	FOR_READ = iota + 1
	FOR_WRITE
)
