package registry

import (
	"strings"
	"testing"

	. "github.com/xregistry/server/common"
)

// TestCanonicalPrettyPrintJSON_SingleRegistry verifies that a single
// Registry entity (given in scrambled/unordered attribute order, with an
// extension attribute mixed in) gets reordered into canonical spec order,
// with the extension attribute placed at the "$extensions" slot.
func TestCanonicalPrettyPrintJSON_SingleRegistry(t *testing.T) {
	input := `{
		"myextension": "hello",
		"name": "Test Registry",
		"registryid": "myreg",
		"epoch": 3,
		"xid": "/",
		"specversion": "1.1",
		"self": "http://example.com/"
	}`

	out, err := CanonicalPrettyPrintJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(out)

	// "registryid" (the real wire-format name for a Registry's generic
	// "id" spec attr) must be recognized and placed right after
	// "specversion" per canonical order.
	if !strings.Contains(got, `"registryid": "myreg"`) {
		t.Errorf("expected registryid recognized as a spec attr, got:\n%s", got)
	}

	// specversion must come before registryid, which must come before
	// self, which must come before epoch, per OrderedSpecProps order.
	idxSpecVersion := strings.Index(got, `"specversion"`)
	idxRegistryID := strings.Index(got, `"registryid"`)
	idxSelf := strings.Index(got, `"self"`)
	idxEpoch := strings.Index(got, `"epoch"`)
	idxName := strings.Index(got, `"name"`)
	idxExt := strings.Index(got, `"myextension"`)
	if !(idxSpecVersion < idxRegistryID && idxRegistryID < idxSelf && idxSelf < idxEpoch && idxEpoch < idxName) {
		t.Errorf("attributes not in canonical order, got:\n%s", got)
	}
	// Extension attribute should come after the known spec attrs (it's
	// alphabetized at the $extensions slot, which follows "labels" in
	// OrderedSpecProps, i.e. after all the attrs present in this test).
	if idxExt < idxName {
		t.Errorf("expected extension attribute after spec attrs, got:\n%s", got)
	}
}

// TestCanonicalPrettyPrintJSON_ExtensionsAlphabetized verifies that
// multiple unrecognized (extension) attributes on an entity are emitted
// in alphabetical order.
func TestCanonicalPrettyPrintJSON_ExtensionsAlphabetized(t *testing.T) {
	input := `{
		"xid": "/",
		"specversion": "1.1",
		"registryid": "myreg",
		"self": "http://example.com/",
		"epoch": 1,
		"zeta": 1,
		"alpha": 2,
		"mike": 3
	}`

	out, err := CanonicalPrettyPrintJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(out)

	idxAlpha := strings.Index(got, `"alpha"`)
	idxMike := strings.Index(got, `"mike"`)
	idxZeta := strings.Index(got, `"zeta"`)
	if idxAlpha < 0 || idxMike < 0 || idxZeta < 0 {
		t.Fatalf("missing expected extension keys in output:\n%s", got)
	}
	if !(idxAlpha < idxMike && idxMike < idxZeta) {
		t.Errorf("extensions not alphabetized, got:\n%s", got)
	}
}

// TestCanonicalPrettyPrintJSON_BlankLines verifies blank-line separators
// are inserted (and collapsed appropriately) between attribute groups per
// OrderedSpecProps' "$space" markers, for a Meta entity (whose "$space"
// marker sits between "$extensions" and "defaultversionid"/etc, unlike
// most other levels where the trailing "$space" ends up adjacent only to
// unresolvable "$RESOURCE*"/"$COLLECTIONS" placeholders and therefore
// never actually renders).
func TestCanonicalPrettyPrintJSON_BlankLines(t *testing.T) {
	input := `{
		"xid": "/groups/g1/resources/r1/meta",
		"readonly": false,
		"compatibility": "none",
		"defaultversionid": "v1",
		"defaultversionurl": "http://example.com/v1",
		"defaultversionsticky": false
	}`

	out, err := CanonicalPrettyPrintJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(out)
	// There should be at least one blank line (an empty line consisting
	// of nothing between two newlines) somewhere in the object body.
	if !strings.Contains(got, "\n\n") {
		t.Errorf("expected at least one blank-line separator, got:\n%s", got)
	}
	// No blank line should appear as the very first or last line of the
	// object body (leading/trailing $space must be trimmed).
	lines := strings.Split(got, "\n")
	if len(lines) < 2 {
		t.Fatalf("unexpected output shape:\n%s", got)
	}
	if strings.TrimSpace(lines[1]) == "" {
		t.Errorf("blank line leaked as first body line:\n%s", got)
	}
	if strings.TrimSpace(lines[len(lines)-2]) == "" {
		t.Errorf("blank line leaked as last body line:\n%s", got)
	}
}

// TestCanonicalPrettyPrintJSON_NestedRegistryDoc verifies the recursive
// structural walk: a full nested Registry doc (registry -> groups
// collection -> group -> resources collection -> resource -> versions
// collection -> version) gets every nested entity's own attributes
// reordered, while collection wrapper maps keep their own key order.
func TestCanonicalPrettyPrintJSON_NestedRegistryDoc(t *testing.T) {
	input := `{
		"myext": true,
		"xid": "/",
		"specversion": "1.1",
		"registryid": "myreg",
		"self": "http://example.com/",
		"epoch": 1,
		"dirscount": 1,
		"dirs": {
			"d1": {
				"description": "a dir",
				"xid": "/dirs/d1",
				"dirid": "d1",
				"self": "http://example.com/dirs/d1",
				"epoch": 2
			}
		}
	}`

	out, err := CanonicalPrettyPrintJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(out)

	// The nested group's own known attrs should be reordered per the
	// Group level's canonical order: self before xid before epoch
	// before description.
	idxXid := strings.LastIndex(got, `"xid": "/dirs/d1"`)
	idxSelf := strings.Index(got, `"self": "http://example.com/dirs/d1"`)
	idxEpoch := strings.Index(got, `"epoch": 2`)
	idxDesc := strings.Index(got, `"description": "a dir"`)
	if idxXid < 0 || idxSelf < 0 || idxEpoch < 0 || idxDesc < 0 {
		t.Fatalf("missing expected nested keys in output:\n%s", got)
	}
	if !(idxSelf < idxXid && idxXid < idxEpoch && idxEpoch < idxDesc) {
		t.Errorf("nested entity attrs not in canonical order, got:\n%s", got)
	}

	// The collection wrapper "dirs" key should retain its original single
	// child key "d1" (i.e. collection map itself isn't reordered/altered
	// beyond recursing into its child).
	if !strings.Contains(got, `"d1": {`) {
		t.Errorf("expected collection child key \"d1\" preserved, got:\n%s", got)
	}
}

// TestCanonicalPrettyPrintJSON_GroupIDResolvedByValue verifies that a
// Group entity's own "<singular>id" attribute (e.g. "dirid") is
// recognized (by matching its VALUE against the entity's own xid
// GroupID) and placed at the canonical "id" slot, rather than falling
// through to being treated as an alphabetized extension.
func TestCanonicalPrettyPrintJSON_GroupIDResolvedByValue(t *testing.T) {
	input := `{
		"self": "http://example.com/dirs/d1",
		"xid": "/dirs/d1",
		"epoch": 1,
		"dirid": "d1",
		"myext": true
	}`
	out, err := CanonicalPrettyPrintJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(out)

	idxDirID := strings.Index(got, `"dirid": "d1"`)
	idxSelf := strings.Index(got, `"self"`)
	idxExt := strings.Index(got, `"myext"`)
	if idxDirID < 0 {
		t.Fatalf("expected \"dirid\" recognized as the resolved id attr, got:\n%s", got)
	}
	// Group level's "id" token is first in the canonical order, so
	// "dirid" must come before "self", and (being a resolved spec attr,
	// not an extension) before "myext" too.
	if !(idxDirID < idxSelf && idxDirID < idxExt) {
		t.Errorf("expected \"dirid\" at the canonical id slot, got:\n%s", got)
	}
}

// TestCanonicalPrettyPrintJSON_VersionResourceFamilyResolved verifies
// that a Version entity's own resource-content attrs (the "$RESOURCE"
// family: <singular>url/<singular>/<singular>base64) are recognized once
// the singular name is derived from the entity's own resolved id
// attribute (e.g. "messageid" -> singular "message"), and placed at the
// canonical "$RESOURCE*" slot instead of falling through to extensions.
func TestCanonicalPrettyPrintJSON_VersionResourceFamilyResolved(t *testing.T) {
	input := `{
		"xid": "/groups/g1/resources/r1/versions/v1",
		"versionid": "v1",
		"self": "http://example.com/v1",
		"epoch": 1,
		"messageid": "r1",
		"messageurl": "http://example.com/r1/content",
		"message": "hello world",
		"messagebase64": "aGVsbG8="
	}`
	out, err := CanonicalPrettyPrintJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(out)

	idxMessageID := strings.Index(got, `"messageid": "r1"`)
	idxVersionID := strings.Index(got, `"versionid"`)
	if idxMessageID < 0 || idxVersionID < 0 {
		t.Fatalf("missing expected keys, got:\n%s", got)
	}
	// The generic "id" token comes before "versionid" in Version's
	// canonical order.
	if !(idxMessageID < idxVersionID) {
		t.Errorf("expected \"messageid\" at the canonical id slot (before versionid), got:\n%s", got)
	}

	idxMessageURL := strings.Index(got, `"messageurl"`)
	idxMessage := strings.Index(got, `"message":`)
	idxMessageBase64 := strings.Index(got, `"messagebase64"`)
	if idxMessageURL < 0 || idxMessage < 0 || idxMessageBase64 < 0 {
		t.Fatalf("missing expected resource-family keys, got:\n%s", got)
	}
	if !(idxMessageURL < idxMessage && idxMessage < idxMessageBase64) {
		t.Errorf("expected resource-family attrs in url/bare/base64 order, got:\n%s", got)
	}
}

// TestCanonicalPrettyPrintJSON_CollectionsHeuristic verifies that
// Registry-level (group-type) collections — which have no fixed name and
// can be repeated any number of times — are detected heuristically (an
// unconsumed "<name>count" key paired with an unconsumed "<name>url"
// sibling) rather than falling through to being treated as unrelated
// alphabetized extensions, and that multiple such collections are
// ordered alphabetically by base name.
func TestCanonicalPrettyPrintJSON_CollectionsHeuristic(t *testing.T) {
	input := `{
		"xid": "/",
		"specversion": "1.1",
		"registryid": "myreg",
		"self": "http://example.com/",
		"epoch": 1,
		"endpointscount": 1,
		"endpointsurl": "http://example.com/endpoints",
		"dirscount": 2,
		"dirsurl": "http://example.com/dirs",
		"myext": true
	}`
	out, err := CanonicalPrettyPrintJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(out)

	idxDirsURL := strings.Index(got, `"dirsurl"`)
	idxDirsCount := strings.Index(got, `"dirscount"`)
	idxEndpointsURL := strings.Index(got, `"endpointsurl"`)
	idxEndpointsCount := strings.Index(got, `"endpointscount"`)
	idxExt := strings.Index(got, `"myext"`)
	if idxDirsURL < 0 || idxDirsCount < 0 || idxEndpointsURL < 0 || idxEndpointsCount < 0 {
		t.Fatalf("missing expected collection keys, got:\n%s", got)
	}
	// Each collection: url before count.
	if !(idxDirsURL < idxDirsCount) {
		t.Errorf("expected dirsurl before dirscount, got:\n%s", got)
	}
	if !(idxEndpointsURL < idxEndpointsCount) {
		t.Errorf("expected endpointsurl before endpointscount, got:\n%s", got)
	}
	// Collections ordered alphabetically by base name: dirs before endpoints.
	if !(idxDirsURL < idxEndpointsURL) {
		t.Errorf("expected collections alphabetized (dirs before endpoints), got:\n%s", got)
	}
	// Collections are recognized spec-level constructs, not extensions,
	// so myext (a genuine, unrelated extension) should still be
	// alphabetized separately and not confused with them.
	if idxExt < 0 {
		t.Fatalf("missing myext, got:\n%s", got)
	}
}

// TestCanonicalPrettyPrintJSON_ResourceVersionsCollectionHardcoded
// verifies that a Resource entity's "versions" collection — always the
// fixed, model-independent "versionsurl"/"versionscount"/"versions"
// triple — is recognized without needing the count/url heuristic (i.e.
// even if "versions" itself, the inlined map, is absent).
func TestCanonicalPrettyPrintJSON_ResourceVersionsCollectionHardcoded(t *testing.T) {
	input := `{
		"xid": "/groups/g1/resources/r1",
		"self": "http://example.com/r1",
		"messageid": "r1",
		"versionscount": 3,
		"versionsurl": "http://example.com/r1/versions",
		"myext": "z"
	}`
	out, err := CanonicalPrettyPrintJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(out)

	idxVersionsURL := strings.Index(got, `"versionsurl"`)
	idxVersionsCount := strings.Index(got, `"versionscount"`)
	idxExt := strings.Index(got, `"myext"`)
	if idxVersionsURL < 0 || idxVersionsCount < 0 || idxExt < 0 {
		t.Fatalf("missing expected keys, got:\n%s", got)
	}
	// versionsurl/versionscount are recognized as the hardcoded
	// "versions" collection (in url-before-count order) and placed at
	// the "$COLLECTIONS" slot, which in Resource's canonical order comes
	// after (a blank-line-separated) "$extensions" — so "myext" (a
	// genuine, unrelated extension) comes first, followed by a blank
	// line, then the collection keys.
	if !(idxExt < idxVersionsURL && idxVersionsURL < idxVersionsCount) {
		t.Errorf("expected myext before versionsurl before versionscount, got:\n%s", got)
	}
	if !strings.Contains(got, "\n\n") {
		t.Errorf("expected a blank line separating extensions from the collection, got:\n%s", got)
	}
}

// TestCanonicalPrettyPrintJSON_ResourceVersionMirroredAttrs verifies that
// a Resource-level entity whose JSON also carries its default Version's
// mirrored attrs (isdefault/createdat/modifiedat/ancestorid/etc. - what
// the server's real GET .../resourceID[$details] response actually looks
// like, since it copies the default Version's own row onto the Resource
// row - see registry/resource.go's SaveDefaultVersionCascade()/
// IsDefaultVerCopy mechanism) gets those Version-typed attrs recognized
// and placed in canonical order too, instead of falling through to being
// alphabetized as plain extensions (which is what happened before the
// Resource-level token list was extended to include ENTITY_VERSION-typed
// attrs alongside its own ENTITY_RESOURCE-typed ones).
func TestCanonicalPrettyPrintJSON_ResourceVersionMirroredAttrs(t *testing.T) {
	input := `{
		"fileid": "f1",
		"versionid": "1",
		"self": "http://example.com/dirs/d1/files/f1$details",
		"xid": "/dirs/d1/files/f1",
		"epoch": 1,
		"isdefault": true,
		"createdat": "2020-01-01T00:00:00Z",
		"modifiedat": "2020-01-01T00:00:00Z",
		"ancestorid": "1",
		"metaurl": "http://example.com/dirs/d1/files/f1/meta",
		"versionsurl": "http://example.com/dirs/d1/files/f1/versions",
		"versionscount": 1
	}`
	out, err := CanonicalPrettyPrintJSON([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := string(out)

	// None of these should have fallen through to being alphabetized
	// extensions - they must all be recognized as spec attrs, in this
	// exact relative order (mirrors OrderedSpecProps' declaration
	// order): id, versionid, self, xid, epoch, isdefault, createdat,
	// modifiedat, ancestorid, metaurl, versionsurl, versionscount.
	names := []string{
		`"fileid"`, `"versionid"`, `"self"`, `"xid"`, `"epoch"`,
		`"isdefault"`, `"createdat"`, `"modifiedat"`, `"ancestorid"`,
		`"metaurl"`, `"versionsurl"`, `"versionscount"`,
	}
	idx := make([]int, len(names))
	for i, n := range names {
		idx[i] = strings.Index(got, n)
		if idx[i] < 0 {
			t.Fatalf("missing expected key %s, got:\n%s", n, got)
		}
	}
	for i := 1; i < len(idx); i++ {
		if idx[i-1] >= idx[i] {
			t.Errorf("expected %s before %s (canonical order), got:\n%s", names[i-1], names[i], got)
		}
	}
}

// TestCanonicalPrettyReorderTree_MinimizeAfterReorder simulates what
// cmds/xr/get.go's minimize() now does for "xr get -m --min": reorder
// the tree FIRST (via CanonicalPrettyReorderTree, while xid/ids are
// still present), THEN delete keys from the already-ordered
// *OrderedMap (via Delete(), which preserves relative order), THEN
// stringify what's left (via StringifyCanonicalTree). This verifies
// that deleting keys AFTER reordering still produces correct canonical
// order and correctly collapsed blank-line spacing for what remains -
// the core fix for the "xr get -m" attribute-ordering bug (deleting
// BEFORE reordering lost the xid/id signals needed to classify/order
// entities at all).
func TestCanonicalPrettyReorderTree_MinimizeAfterReorder(t *testing.T) {
	input := `{
		"fileid": "f1",
		"versionid": "1",
		"self": "http://example.com/dirs/d1/files/f1$details",
		"xid": "/dirs/d1/files/f1",
		"epoch": 1,
		"isdefault": true,
		"createdat": "2020-01-01T00:00:00Z",
		"modifiedat": "2020-01-01T00:00:00Z",
		"ancestorid": "1",
		"metaurl": "http://example.com/dirs/d1/files/f1/meta",
		"versionsurl": "http://example.com/dirs/d1/files/f1/versions",
		"versionscount": 1
	}`

	tree, err := CanonicalPrettyReorderTree([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	om, ok := tree.(*OrderedMap)
	if !ok {
		t.Fatalf("expected *OrderedMap, got %T", tree)
	}

	// Simulate minimize()'s deletions for a single-version Resource
	// (rm.GetMaxVersions() == 1 branch): strip xid/self plus the
	// id/version/collection bookkeeping attrs, same as cmds/xr/get.go.
	for _, k := range []string{
		"xid", "self", "fileid", "versionid", "isdefault",
		"versionscount", "versionsurl", "ancestorid", "metaurl",
	} {
		om.Delete(k)
	}

	out, err := StringifyCanonicalTree(om)
	if err != nil {
		t.Fatalf("unexpected error stringifying: %v", err)
	}
	got := string(out)

	// What's left (epoch/createdat/modifiedat) must still appear in
	// canonical order, with no leftover/duplicated blank lines from the
	// now-empty gaps where deleted keys used to be.
	idxEpoch := strings.Index(got, `"epoch"`)
	idxCreated := strings.Index(got, `"createdat"`)
	idxModified := strings.Index(got, `"modifiedat"`)
	if idxEpoch < 0 || idxCreated < 0 || idxModified < 0 {
		t.Fatalf("missing expected keys after minimize, got:\n%s", got)
	}
	if !(idxEpoch < idxCreated && idxCreated < idxModified) {
		t.Errorf("expected epoch < createdat < modifiedat, got:\n%s", got)
	}
	if strings.Contains(got, "\n\n") {
		t.Errorf("expected no blank lines left (all separated sections were fully deleted), got:\n%s", got)
	}
	for _, deleted := range []string{"xid", "self", "fileid", "versionid", "isdefault", "versionscount", "versionsurl", "ancestorid", "metaurl"} {
		if strings.Contains(got, `"`+deleted+`"`) {
			t.Errorf("expected %q to be deleted, got:\n%s", deleted, got)
		}
	}
}

// TestCanonicalPrettyReorderTree_MinimizeNestedDocCollapsesBlanks
// verifies that minimize()-style deletion on a nested Registry doc
// (reordered first) still leaves a correctly collapsed single blank
// line between sections when some (but not all) of the keys around a
// blank-line boundary get deleted.
func TestCanonicalPrettyReorderTree_MinimizeNestedDocCollapsesBlanks(t *testing.T) {
	input := `{
		"xid": "/",
		"specversion": "1.1",
		"registryid": "myreg",
		"self": "http://example.com/",
		"epoch": 1,
		"dirscount": 1,
		"dirsurl": "http://example.com/dirs",
		"dirs": {
			"d1": {
				"xid": "/dirs/d1",
				"dirid": "d1",
				"self": "http://example.com/dirs/d1",
				"epoch": 2
			}
		}
	}`

	tree, err := CanonicalPrettyReorderTree([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	om, ok := tree.(*OrderedMap)
	if !ok {
		t.Fatalf("expected *OrderedMap, got %T", tree)
	}

	// Minimize the Registry level: drop xid/self/specversion/registryid
	// and the "dirs" collection's own url/count (mirrors minimize()'s
	// ENTITY_REGISTRY case), but keep the "dirs" map itself and recurse
	// into its child the same way minimize() does.
	for _, k := range []string{"xid", "self", "specversion", "registryid", "dirsurl", "dirscount"} {
		om.Delete(k)
	}
	dirsAny := om.Get("dirs")
	dirs, ok := dirsAny.(*OrderedMap)
	if !ok {
		t.Fatalf("expected \"dirs\" to be *OrderedMap, got %T", dirsAny)
	}
	d1Any := dirs.Get("d1")
	d1, ok := d1Any.(*OrderedMap)
	if !ok {
		t.Fatalf("expected \"d1\" to be *OrderedMap, got %T", d1Any)
	}
	for _, k := range []string{"xid", "self", "dirid"} {
		d1.Delete(k)
	}

	out, err := StringifyCanonicalTree(om)
	if err != nil {
		t.Fatalf("unexpected error stringifying: %v", err)
	}
	got := string(out)

	idxEpoch := strings.Index(got, `"epoch": 1`)
	idxDirs := strings.Index(got, `"dirs"`)
	idxD1Epoch := strings.Index(got, `"epoch": 2`)
	if idxEpoch < 0 || idxDirs < 0 || idxD1Epoch < 0 {
		t.Fatalf("missing expected keys after minimize, got:\n%s", got)
	}
	// Registry-level: "epoch" then a single blank line then "dirs" (the
	// dirsurl/dirscount that used to sit between them are gone, but the
	// blank-line separator immediately preceding them in canonical order
	// should still collapse to exactly one, not zero or several).
	if !(idxEpoch < idxDirs) {
		t.Errorf("expected epoch before dirs, got:\n%s", got)
	}
	if !strings.Contains(got, "\n\n") {
		t.Errorf("expected exactly one blank line to survive between epoch and dirs, got:\n%s", got)
	}
	if strings.Count(got, "\n\n\n") > 0 {
		t.Errorf("expected no double-blank-line artifacts, got:\n%s", got)
	}
	// Nested d1's own remaining key ("epoch": 2) must not have picked up
	// a stray leading blank line now that xid/self/dirid are all gone.
	if strings.Contains(got, "{\n\n") {
		t.Errorf("expected no leading blank line inside an object, got:\n%s", got)
	}
}

// TestOrderedMap_RealKeyCountIgnoresBlankSentinels verifies that
// deleting every real attribute from a canonically-reordered entity
// (e.g. what cmds/xr/download.go's makeImportObj does for a
// single-version Resource under --min) can leave lingering blank-line
// sentinel keys behind - since Delete() only ever targets named
// attributes, never the reserved blank-line separator - and that
// RealKeyCount() (unlike plain len(om.Keys)) correctly reports zero
// real content remains in that case. This is the fix for a regression
// where "xr download --min" wrote out spurious "{}" files for entities
// that should have been skipped entirely, because len(om.Keys) > 0
// was true (due to leftover sentinels) even though the entity had no
// real attributes left.
func TestOrderedMap_RealKeyCountIgnoresBlankSentinels(t *testing.T) {
	input := `{
		"fileid": "f1",
		"versionid": "1",
		"self": "http://example.com/dirs/d1/files/f1$details",
		"xid": "/dirs/d1/files/f1",
		"epoch": 1,
		"isdefault": true,
		"createdat": "2020-01-01T00:00:00Z",
		"modifiedat": "2020-01-01T00:00:00Z",
		"ancestorid": "1",
		"metaurl": "http://example.com/dirs/d1/files/f1/meta",
		"versionsurl": "http://example.com/dirs/d1/files/f1/versions",
		"versionscount": 1
	}`

	tree, err := CanonicalPrettyReorderTree([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	om, ok := tree.(*OrderedMap)
	if !ok {
		t.Fatalf("expected *OrderedMap, got %T", tree)
	}

	// Delete every real attribute, same as makeImportObj()/minimize()
	// would for a fully-stripped single-version Resource.
	for _, k := range []string{
		"xid", "self", "fileid", "versionid", "isdefault",
		"versionscount", "versionsurl", "ancestorid", "metaurl",
		"epoch", "createdat", "modifiedat",
	} {
		om.Delete(k)
	}

	if got := om.RealKeyCount(); got != 0 {
		t.Errorf("expected RealKeyCount() == 0 after deleting all real attrs, got %d (om.Keys=%v)", got, om.Keys)
	}
	if len(om.Keys) == 0 {
		t.Errorf("expected len(om.Keys) to still be > 0 here (lingering blank sentinels) - if this now fails, the sentinel-leftover scenario this test guards may no longer reproduce, please double check RealKeyCount() is still needed")
	}

	out, err := StringifyCanonicalTree(om)
	if err != nil {
		t.Fatalf("unexpected error stringifying: %v", err)
	}
	if string(out) != "{}" {
		t.Errorf("expected stringified output to collapse to exactly \"{}\", got %q", string(out))
	}
}

// TestCanonicalPrettyReorderTreeOrRaw verifies the rawjson-mode helper
// used by the "xr" CLI (`.xr` config option "rawjson"): with rawJSON
// false it must behave identically to CanonicalPrettyReorderTree
// (canonical order, blank-line sentinels inserted); with rawJSON true
// it must preserve the input's original key order verbatim and insert
// no blank-line sentinels at all.
func TestCanonicalPrettyReorderTreeOrRaw(t *testing.T) {
	input := `{
		"description": "out of canonical order on purpose",
		"dirid": "d1",
		"xid": "/dirs/d1",
		"self": "http://example.com/dirs/d1",
		"epoch": 1,
		"name": "D1"
	}`

	// rawJSON=false: same result as CanonicalPrettyReorderTree.
	wantTree, err := CanonicalPrettyReorderTree([]byte(input))
	if err != nil {
		t.Fatalf("CanonicalPrettyReorderTree unexpected error: %v", err)
	}
	wantOut, err := StringifyCanonicalTree(wantTree)
	if err != nil {
		t.Fatalf("unexpected error stringifying want tree: %v", err)
	}

	gotTree, err := CanonicalPrettyReorderTreeOrRaw([]byte(input), false)
	if err != nil {
		t.Fatalf("CanonicalPrettyReorderTreeOrRaw(false) unexpected error: %v", err)
	}
	gotOut, err := StringifyCanonicalTree(gotTree)
	if err != nil {
		t.Fatalf("unexpected error stringifying got tree: %v", err)
	}
	if string(gotOut) != string(wantOut) {
		t.Errorf("rawJSON=false should match CanonicalPrettyReorderTree exactly\nwant:\n%s\ngot:\n%s", wantOut, gotOut)
	}

	// rawJSON=true: original key order preserved, no blank lines.
	rawTree, err := CanonicalPrettyReorderTreeOrRaw([]byte(input), true)
	if err != nil {
		t.Fatalf("CanonicalPrettyReorderTreeOrRaw(true) unexpected error: %v", err)
	}
	rawOm, ok := rawTree.(*OrderedMap)
	if !ok {
		t.Fatalf("expected *OrderedMap, got %T", rawTree)
	}

	wantKeyOrder := []string{"description", "dirid", "xid", "self", "epoch", "name"}
	if len(rawOm.Keys) != len(wantKeyOrder) {
		t.Fatalf("expected %d keys preserved in original order, got %d (%v)", len(wantKeyOrder), len(rawOm.Keys), rawOm.Keys)
	}
	for i, k := range wantKeyOrder {
		if rawOm.Keys[i] != k {
			t.Errorf("key[%d]: expected %q, got %q (full order: %v)", i, k, rawOm.Keys[i], rawOm.Keys)
		}
	}

	rawOut, err := StringifyCanonicalTree(rawTree)
	if err != nil {
		t.Fatalf("unexpected error stringifying raw tree: %v", err)
	}
	if strings.Contains(string(rawOut), "\n\n") {
		t.Errorf("rawJSON=true output should have no blank-line separators, got:\n%s", rawOut)
	}
}
