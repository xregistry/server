package registry

import "os"

var TESTING = (os.Getenv("TESTING") != "")
var GitCommit = "<n/a>"

var MAX_VARCHAR = 1024

const SPECVERSION = "1.0-rc1"
const XREGSCHEMA = "xRegistry-json"

// Model attribute default values
const STRICT = true
const MAXVERSIONS = 0
const SETVERSIONID = true
const SETDEFAULTSTICKY = true
const HASDOCUMENT = true
const SINGLEVERSIONROOT = false
const READONLY = false

// Attribute types
const ANY = "any"
const ARRAY = "array"
const BOOLEAN = "boolean"
const DECIMAL = "decimal"
const INTEGER = "integer"
const MAP = "map"
const OBJECT = "object"
const XID = "xid"
const STRING = "string"
const TIMESTAMP = "timestamp"
const UINTEGER = "uinteger"
const URI = "uri"
const URI_REFERENCE = "urireference"
const URI_TEMPLATE = "uritemplate"
const URL = "url"

const IN_CHAR = '.'
const IN_STR = string(IN_CHAR)

const UX_IN = '.'

// If DB_IN changes then DefaultProps in init.sql needs to change too
const DB_IN = ','
const DB_INDEX = '#'

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
	// NEVER ADD TO THE MIDDLE - ALWAYS TO THE END
	ENTITY_NONE     = -1
	ENTITY_REGISTRY = iota
	ENTITY_GROUP
	ENTITY_RESOURCE
	ENTITY_META
	ENTITY_VERSION
	ENTITY_MODEL
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
