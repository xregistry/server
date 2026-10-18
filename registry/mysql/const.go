package mysql

var MAX_VARCHAR = 4096
var MAX_PROPNAME = 255

// COLLATE clause for case-insensitive string comparisons per spec
const FILTER_CI_COLLATE = "COLLATE utf8mb4_0900_ai_ci"
