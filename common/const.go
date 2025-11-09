package common

var GitCommit = ""

const SPECVERSION = "1.0-rc2"
const SPECURL = "https://github.com/xregistry/spec/blob/main/core/spec.md"

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
const XIDTYPE = "xidtype"
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
