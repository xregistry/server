# xRegistry Server Implementation Guide

**⚠️ READ THE SPEC FIRST: https://github.com/xregistry/spec**

The xRegistry specification is the authoritative source for API behavior, entity relationships, attribute definitions, and all protocol details. This document only supplements the spec with implementation-specific patterns and gotchas.

---

This document describes key patterns and design decisions in the xRegistry server. Some patterns are mandated by the [xRegistry specification](https://github.com/xregistry/spec), while others are implementation-specific. Both are documented here for quick reference.

## Purpose

Quick reference for developers working on this codebase, covering:
- **Spec-mandated behaviors** that might not be obvious from code alone
- **Implementation-specific patterns** unique to this MySQL-based server
- **Database conventions** and gotchas
- **Testing patterns** and requirements

**When in doubt, consult the spec!** This document exists to avoid constantly re-reading the spec for common patterns, but the spec always wins in case of conflict.

---

## Entity Creation Patterns (from spec)

### Implicit Parent/Child Creation

The xRegistry spec mandates that the server automatically creates parent and child entities when needed.

**Creating versions auto-creates parents:**
```bash
# This single request creates group "d1", resource "f1", AND version "v99"
PUT /dirs/d1/files/f1/versions/v99
{
  "description": "My version"
}
```

The server will:
1. Check if group "d1" exists; if not, create it
2. Check if resource "f1" exists; if not, create it  
3. Create version "v99"

**Creating resources auto-creates parents:**
```bash
# Creates group "d1" and resource "f1"
# Also auto-creates version "1" since no version specified
PUT /dirs/d1/files/f1
{
  "description": "My file"
}
```

**Key points:**
- ✅ Parents (groups, resources) are auto-created when creating children
- ✅ When creating a resource without specifying version, version "1" is auto-created
- ❌ Creating a version does NOT auto-create siblings (other versions)

**Code Location:** `registry/resource.go` - `UpsertResource` calls `FindOrAddGroup`; `UpsertVersionWithObject` handles implicit version creation

**Why this matters:** Tests don't need to explicitly create parent entities. One PUT can create the entire hierarchy.

---

## Model Changes

As of now, removing a Group or Resource type will silently delete any of the
corresponding entity instances of those types.

However, attribute-level changes that make any entity in the system be
non-compliant will generate an error and the model update will be rejected.

---

## Model Auto-Generation (from spec)

### Document Attributes

When `hasdocument=true` (the spec default for resources), the following attributes become automatically available without being explicitly defined in the model:

- `<singular>` - The document content itself (e.g., `file`)
- `<singular>url` - External document URL (e.g., `fileurl`)
- `<singular>proxyurl` - Proxied document URL (e.g., `fileproxyurl`)  
- `<singular>base64` - Base64-encoded document (e.g., `filebase64`)

**Example:**
```json
// Minimal model for a "files" resource:
{
  "resources": {
    "files": {
      "singular": "file",
      "plural": "files"
      // hasdocument defaults to true
    }
  }
}

// Users can now provide these attributes in requests:
PUT /dirs/d1/files/f1/versions/1$details
{
  "fileurl": "http://example.com/doc.txt"  // Auto-available!
}
```

**When `hasdocument=false`:** These attributes are NOT auto-generated. Users can manually define attributes with these names in the model if needed, but they become regular user-defined attributes with no special document-handling behavior.

**Code Location:** `registry/shared_model.go:1261-1267` - `GetPropsOrdered()` transforms `$RESOURCE*` template attributes based on `hasdocument` setting

---

## Database Schema Conventions (implementation-specific)

### PropName Includes Delimiter (DB_IN)

**Critical Implementation Detail:** In the `Props` table, the `PropName` column ALWAYS includes a trailing delimiter character (`DB_IN`, which is a comma `,`).

**Examples:**
- Stored as: `"fileurl,"` (note the comma)
- NOT: `"fileurl"`

**Why?** Simplifies parsing and prevents ambiguity (e.g., distinguishing `"file,"` from `"fileurl,"`).

**When Querying Props:**
```sql
-- CORRECT: Include DB_IN delimiter
WHERE PropName = 'fileurl' || string(DB_IN)
WHERE PropName IN ('file,', 'fileurl,', 'fileproxyurl,')

-- WRONG: Will never match any rows
WHERE PropName = 'fileurl'
WHERE PropName IN ('file', 'fileurl', 'fileproxyurl')
```

**Code Locations:**
- `common/const.go:50` - `DB_IN` constant definition (`const DB_IN = ','`)
- `registry/init.sql:196-217` - Props table definition with detailed comment
- `registry/resource.go:1174-1209` - Example of correct DB_IN usage in hasdocument violation check

**This is the #1 source of bugs when writing new SQL queries.** Always append `DB_IN` when filtering by PropName!

### Entity Identifiers: SID vs UID

- **SID (System ID)**: Auto-generated, globally unique. Used for all database JOINs and foreign keys.
- **UID (User ID)**: User-provided from API (e.g., resource ID, version ID). Only unique within parent scope.

**Code Location:** Comment at top of `registry/init.sql` lines 11-22

---

## Validation Order (implementation-specific)

When a single request updates both the model AND entity attributes (e.g., `PUT /` with `modelsource` attribute), the validation order matters.

### Correct Flow

1. **Apply model changes** (save to DB)
2. **Apply entity attribute changes**  
3. **Save entity changes**
4. **Validate all data** against the new model

### Why This Matters

```json
// Current state: description="OnE"
PUT /
{
  "description": "TWO",
  "modelsource": {
    "attributes": {
      "description": {
        "enum": ["oNe", "TWO"],
        "matchcase": true
      }
    }
  }
}
```

- ❌ **Wrong order:** Apply model → Validate → Apply attributes = Fails (old "OnE" invalid under new enum)
- ✅ **Right order:** Apply model → Apply attributes → Validate = Succeeds (new "TWO" valid under new enum)

### Implementation

Uses `registry.SetStuff("modelchanged", true)` to track when model changed, then calls `VerifyData()` after attribute updates instead of during model application.

**Code Locations:**
- `registry/info.go:1104-1111` - Sets flag when model applied  
- `registry/registry.go:587-603` - Checks flag and calls `VerifyData()`

---

## Test Patterns (project conventions)

### Byte-for-Byte Comparison

Tests compare exact JSON output, not wildcards.

```go
// GOOD: Exact verification
XHTTP(t, reg, "PUT", "/dirs/d1", `{}`, 201, `{
  "dirid": "d1",
  "epoch": 1,
  ...
}`)

// AVOID: Hides regressions
XHTTP(t, reg, "PUT", "/dirs/d1", `{}`, 201, `*`)
```

**Exceptions:** Timestamps and hostnames are auto-masked by test framework.

### Timestamp Masking Pattern

Test framework masks timestamps with pattern `YYYY-MM-DDTHH:MM:SSZ` where:
- Date/time portions (YYYY-MM-DDTHH:MM) are masked as literal strings
- Seconds (SS) start at `:01Z` and increment for each operation: `:01Z`, `:02Z`, `:03Z`, etc.

```go
// First operation
"createdat": "YYYY-MM-DDTHH:MM:01Z",
"modifiedat": "YYYY-MM-DDTHH:MM:01Z"

// Second operation (PATCH)
"modifiedat": "YYYY-MM-DDTHH:MM:02Z"  // Seconds increment

// Third operation
"modifiedat": "YYYY-MM-DDTHH:MM:03Z"  // And so on
```

**Key Points:**
- Timestamps start at `:01Z` (not `:00Z`)
- Each write operation increments the seconds counter
- Read operations don't affect the counter
- Epoch can advance without timestamp changing if operations happen quickly

### No Epoch Masks

Per user requirement: Always verify `epoch` increments correctly. Do NOT use `epochMask` to hide epoch values.

**Code Location:** `tests/model3_test.go` - Examples with explicit epoch values in expected output

### Testing Resources with hasDocument=true

Resources with `hasdocument=true` have two access patterns:
- **Document view:** Direct resource URL returns the document content
- **Metadata view:** Use `$details` endpoint to access xRegistry metadata

```go
// WRONG: Can't PATCH metadata directly on hasDocument=true resource
XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1", `{"description": "x"}`, ...)

// CORRECT: Use $details for metadata operations
XHTTP(t, reg, "PATCH", "/dirs/d1/files/f1$details", `{"description": "x"}`, ...)

// TESTING TIP: Set hasDocument=false in test models to simplify tests
_, err = gm.AddResourceModel("files", "file", 0, true, true, false)  // false = hasDocument
```

**Code Location:** `tests/capabilities_test.go` - Examples using hasDocument=false for simpler tests

---

## Quick Reference

### Before Writing SQL Queries

- [ ] Appended `DB_IN` to PropName filters?
- [ ] Used SID (not UID) for JOINs?

### Before Changing Validation Logic

- [ ] Ensured model changes applied before validation?
- [ ] Tracked model changes if validation deferred?

### Before Writing Tests

- [ ] Using byte-for-byte comparison (not wildcards)?
- [ ] Verifying epoch increments explicitly?
- [ ] Relying on implicit parent/child creation (no manual setup)?

---

## Contributing

1. **Check this document first** for patterns that aren't obvious from code
2. **Update this document** when introducing new patterns
3. **Add code comments** referencing this doc at critical decision points
4. **Verify spec compliance** for API behavior changes

If code contradicts this document, the code is probably correct - update the document!
