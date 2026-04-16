package registry

import "testing"

// ── IsValidXMLSchema ──────────────────────────────────────────────

func TestIsValidXMLSchema(t *testing.T) {
	tests := []struct {
		name    string
		schema  string
		wantErr bool
	}{
		{
			name: "minimal schema",
			schema: `<xs:schema ` +
				`xmlns:xs="http://www.w3.org/2001/XMLSchema">` +
				`</xs:schema>`,
		},
		{
			name: "schema with element and complex type",
			schema: `<xs:schema ` +
				`xmlns:xs="http://www.w3.org/2001/XMLSchema">` +
				`<xs:element name="user" type="UserType"/>` +
				`<xs:complexType name="UserType">` +
				`<xs:sequence>` +
				`<xs:element name="id" type="xs:string"/>` +
				`</xs:sequence></xs:complexType></xs:schema>`,
		},
		{
			name:    "not xml",
			schema:  `{"type":"object"}`,
			wantErr: true,
		},
		{
			name: "wrong root",
			schema: `<root ` +
				`xmlns:xs="http://www.w3.org/2001/XMLSchema">` +
				`</root>`,
			wantErr: true,
		},
		{
			name: "wrong namespace",
			schema: `<xs:schema ` +
				`xmlns:xs="http://example.com/ns">` +
				`</xs:schema>`,
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := IsValidXMLSchema([]byte(tc.schema))
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// ── checkXSDCompat helpers ────────────────────────────────────────

func runXSDCompat(
	t *testing.T, dir, old, new string, wantErr bool,
) {
	t.Helper()
	err := checkXSDCompat(dir, []byte(old), []byte(new))
	if wantErr && err == nil {
		t.Error("expected incompatibility, got nil")
	}
	if !wantErr && err != nil {
		t.Errorf("unexpected incompatibility: %v", err)
	}
}

// xs() is a shorthand for the xs: schema prefix declaration.
const xsNS = `xmlns:xs="http://www.w3.org/2001/XMLSchema"`

func xsd(body string) string {
	return `<xs:schema ` + xsNS + `>` + body + `</xs:schema>`
}

// ── Top-level declaration tests ───────────────────────────────────

func TestXSDTopLevelDeclarations(t *testing.T) {
	base := xsd(
		`<xs:element name="id" type="xs:string"/>` +
			`<xs:complexType name="UserType">` +
			`<xs:sequence>` +
			`<xs:element name="name" type="xs:string"/>` +
			`</xs:sequence></xs:complexType>`,
	)
	additive := xsd(
		`<xs:element name="id" type="xs:string"/>` +
			`<xs:element name="email" type="xs:string"/>` +
			`<xs:complexType name="UserType">` +
			`<xs:sequence>` +
			`<xs:element name="name" type="xs:string"/>` +
			`</xs:sequence></xs:complexType>`,
	)
	changed := xsd(
		`<xs:element name="id" type="xs:int"/>` +
			`<xs:complexType name="UserType">` +
			`<xs:sequence>` +
			`<xs:element name="name" type="xs:string"/>` +
			`</xs:sequence></xs:complexType>`,
	)
	removed := xsd(
		`<xs:complexType name="UserType">` +
			`<xs:sequence>` +
			`<xs:element name="name" type="xs:string"/>` +
			`</xs:sequence></xs:complexType>`,
	)

	tests := []struct {
		name    string
		dir     string
		old     string
		new     string
		wantErr bool
	}{
		{"same schema backward", "backward", base, base, false},
		{"same schema forward", "forward", base, base, false},
		{
			"add declaration backward",
			"backward", base, additive, false,
		},
		{
			"remove declaration backward",
			"backward", base, removed, true,
		},
		{
			"change type backward",
			"backward", base, changed, true,
		},
		{
			"add declaration forward",
			"forward", base, additive, true,
		},
		{
			"remove declaration forward",
			"forward", base, removed, false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runXSDCompat(t, tc.dir, tc.old, tc.new, tc.wantErr)
		})
	}
}

// ── Sequence element tests ────────────────────────────────────────

func TestXSDSequenceElements(t *testing.T) {
	withSeq := func(seq string) string {
		return xsd(`<xs:complexType name="T">` +
			`<xs:sequence>` + seq + `</xs:sequence>` +
			`</xs:complexType>`)
	}

	base := withSeq(
		`<xs:element name="a" type="xs:string"/>` +
			`<xs:element name="b" type="xs:int"/>`,
	)
	addOptional := withSeq(
		`<xs:element name="a" type="xs:string"/>` +
			`<xs:element name="b" type="xs:int"/>` +
			`<xs:element name="c" type="xs:string"` +
			` minOccurs="0"/>`,
	)
	addRequired := withSeq(
		`<xs:element name="a" type="xs:string"/>` +
			`<xs:element name="b" type="xs:int"/>` +
			`<xs:element name="c" type="xs:string"/>`,
	)
	removeElem := withSeq(
		`<xs:element name="a" type="xs:string"/>`,
	)
	changeType := withSeq(
		`<xs:element name="a" type="xs:string"/>` +
			`<xs:element name="b" type="xs:string"/>`,
	)
	insertOptBefore := withSeq(
		`<xs:element name="x" type="xs:string"` +
			` minOccurs="0"/>` +
			`<xs:element name="a" type="xs:string"/>` +
			`<xs:element name="b" type="xs:int"/>`,
	)
	insertReqBefore := withSeq(
		`<xs:element name="x" type="xs:string"/>` +
			`<xs:element name="a" type="xs:string"/>` +
			`<xs:element name="b" type="xs:int"/>`,
	)

	tests := []struct {
		name    string
		dir     string
		old     string
		new     string
		wantErr bool
	}{
		{
			"add optional element backward",
			"backward", base, addOptional, false,
		},
		{
			"add required element backward",
			"backward", base, addRequired, true,
		},
		{
			"remove element backward",
			"backward", base, removeElem, true,
		},
		{
			"change element type backward",
			"backward", base, changeType, true,
		},
		{
			"insert optional before backward",
			"backward", base, insertOptBefore, false,
		},
		{
			"insert required before backward",
			"backward", base, insertReqBefore, true,
		},
		// forward: swap writer/reader
		{
			"add optional element forward",
			"forward", base, addOptional, true,
		},
		{
			"add required element forward",
			"forward", base, addRequired, true,
		},
		{
			"remove optional element forward",
			"forward", addOptional, base, false,
		},
		{
			"remove required element forward",
			"forward", base, removeElem, true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runXSDCompat(t, tc.dir, tc.old, tc.new, tc.wantErr)
		})
	}
}

// ── Occurrence constraint tests ───────────────────────────────────

func TestXSDOccurrences(t *testing.T) {
	withElem := func(occurs string) string {
		return xsd(`<xs:complexType name="T">` +
			`<xs:sequence>` +
			`<xs:element name="a" type="xs:string"` +
			occurs + `/>` +
			`</xs:sequence></xs:complexType>`)
	}

	tests := []struct {
		name    string
		dir     string
		old     string
		new     string
		wantErr bool
	}{
		{
			"minOccurs decrease backward (OK)",
			"backward",
			withElem(` minOccurs="2"`),
			withElem(` minOccurs="1"`),
			false,
		},
		{
			"minOccurs increase backward (FAIL)",
			"backward",
			withElem(` minOccurs="1"`),
			withElem(` minOccurs="2"`),
			true,
		},
		{
			"maxOccurs increase backward (OK)",
			"backward",
			withElem(` maxOccurs="2"`),
			withElem(` maxOccurs="5"`),
			false,
		},
		{
			"maxOccurs decrease backward (FAIL)",
			"backward",
			withElem(` maxOccurs="5"`),
			withElem(` maxOccurs="2"`),
			true,
		},
		{
			"maxOccurs to unbounded backward (OK)",
			"backward",
			withElem(` maxOccurs="5"`),
			withElem(` maxOccurs="unbounded"`),
			false,
		},
		{
			"maxOccurs from unbounded backward (FAIL)",
			"backward",
			withElem(` maxOccurs="unbounded"`),
			withElem(` maxOccurs="5"`),
			true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runXSDCompat(t, tc.dir, tc.old, tc.new, tc.wantErr)
		})
	}
}

// ── Attribute tests ───────────────────────────────────────────────

func TestXSDAttributes(t *testing.T) {
	withAttrs := func(attrs string) string {
		return xsd(`<xs:complexType name="T">` +
			`<xs:sequence>` +
			`<xs:element name="x" type="xs:string"/>` +
			`</xs:sequence>` + attrs + `</xs:complexType>`)
	}

	base := withAttrs(
		`<xs:attribute name="id" type="xs:string"` +
			` use="required"/>`,
	)
	addOptAttr := withAttrs(
		`<xs:attribute name="id" type="xs:string"` +
			` use="required"/>` +
			`<xs:attribute name="tag" type="xs:string"` +
			` use="optional"/>`,
	)
	addReqAttr := withAttrs(
		`<xs:attribute name="id" type="xs:string"` +
			` use="required"/>` +
			`<xs:attribute name="tag" type="xs:string"` +
			` use="required"/>`,
	)
	removeAttr := withAttrs(``)
	changeAttrType := withAttrs(
		`<xs:attribute name="id" type="xs:int"` +
			` use="required"/>`,
	)
	reqToOpt := withAttrs(
		`<xs:attribute name="id" type="xs:string"` +
			` use="optional"/>`,
	)
	optToReq := withAttrs(
		`<xs:attribute name="id" type="xs:string"` +
			` use="required"/>`,
	)
	baseOpt := withAttrs(
		`<xs:attribute name="id" type="xs:string"` +
			` use="optional"/>`,
	)

	tests := []struct {
		name    string
		dir     string
		old     string
		new     string
		wantErr bool
	}{
		{
			"add optional attribute backward (OK)",
			"backward", base, addOptAttr, false,
		},
		{
			"add required attribute backward (FAIL)",
			"backward", base, addReqAttr, true,
		},
		{
			"remove attribute backward (FAIL)",
			"backward", base, removeAttr, true,
		},
		{
			"change attribute type backward (FAIL)",
			"backward", base, changeAttrType, true,
		},
		{
			"required to optional backward (OK)",
			"backward", base, reqToOpt, false,
		},
		{
			"optional to required backward (FAIL)",
			"backward", baseOpt, optToReq, true,
		},
		// forward
		{
			"add optional attribute forward (FAIL)",
			"forward", base, addOptAttr, true,
		},
		{
			"remove optional attribute forward (OK)",
			"forward", addOptAttr, base, false,
		},
		{
			"remove required attribute forward (FAIL)",
			"forward", base, removeAttr, true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runXSDCompat(t, tc.dir, tc.old, tc.new, tc.wantErr)
		})
	}
}

// ── simpleType tests ──────────────────────────────────────────────

func TestXSDSimpleType(t *testing.T) {
	withST := func(body string) string {
		return xsd(`<xs:simpleType name="S">` +
			body + `</xs:simpleType>`)
	}
	restr := func(facets string) string {
		return withST(
			`<xs:restriction base="xs:string">` +
				facets + `</xs:restriction>`)
	}

	baseEnum := restr(
		`<xs:enumeration value="A"/>` +
			`<xs:enumeration value="B"/>`,
	)
	addEnum := restr(
		`<xs:enumeration value="A"/>` +
			`<xs:enumeration value="B"/>` +
			`<xs:enumeration value="C"/>`,
	)
	removeEnum := restr(`<xs:enumeration value="A"/>`)
	changeBase := withST(
		`<xs:restriction base="xs:integer">` +
			`<xs:enumeration value="1"/>` +
			`</xs:restriction>`,
	)
	withMaxLen5 := restr(`<xs:maxLength value="5"/>`)
	withMaxLen10 := restr(`<xs:maxLength value="10"/>`)
	withMinLen2 := restr(`<xs:minLength value="2"/>`)
	withMinLen5 := restr(`<xs:minLength value="5"/>`)

	tests := []struct {
		name    string
		dir     string
		old     string
		new     string
		wantErr bool
	}{
		{
			"add enumeration backward (OK)",
			"backward", baseEnum, addEnum, false,
		},
		{
			"remove enumeration backward (FAIL)",
			"backward", baseEnum, removeEnum, true,
		},
		{
			"change base backward (FAIL)",
			"backward", baseEnum, changeBase, true,
		},
		{
			"maxLength increase backward (OK)",
			"backward", withMaxLen5, withMaxLen10, false,
		},
		{
			"maxLength decrease backward (FAIL)",
			"backward", withMaxLen10, withMaxLen5, true,
		},
		{
			"minLength decrease backward (OK)",
			"backward", withMinLen5, withMinLen2, false,
		},
		{
			"minLength increase backward (FAIL)",
			"backward", withMinLen2, withMinLen5, true,
		},
		// forward
		{
			"add enumeration forward (FAIL)",
			"forward", baseEnum, addEnum, true,
		},
		{
			"remove enumeration forward (OK)",
			"forward", baseEnum, removeEnum, false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runXSDCompat(t, tc.dir, tc.old, tc.new, tc.wantErr)
		})
	}
}

// ── complexType extension tests ───────────────────────────────────

func TestXSDComplexContentExtension(t *testing.T) {
	ext := func(base, seq string) string {
		return xsd(`<xs:complexType name="T">` +
			`<xs:complexContent>` +
			`<xs:extension base="` + base + `">` +
			`<xs:sequence>` + seq + `</xs:sequence>` +
			`</xs:extension></xs:complexContent>` +
			`</xs:complexType>`)
	}

	base := ext("BaseType",
		`<xs:element name="extra" type="xs:string"` +
			` minOccurs="0"/>`)
	changeBase := ext("OtherType",
		`<xs:element name="extra" type="xs:string"` +
			` minOccurs="0"/>`)
	addRequired := ext("BaseType",
		`<xs:element name="extra" type="xs:string"` +
			` minOccurs="0"/>` +
			`<xs:element name="must" type="xs:string"/>`)

	tests := []struct {
		name    string
		dir     string
		old     string
		new     string
		wantErr bool
	}{
		{
			"same extension backward (OK)",
			"backward", base, base, false,
		},
		{
			"change base type backward (FAIL)",
			"backward", base, changeBase, true,
		},
		{
			"add required element in extension backward (FAIL)",
			"backward", base, addRequired, true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runXSDCompat(t, tc.dir, tc.old, tc.new, tc.wantErr)
		})
	}
}

// ── group and attributeGroup tests ───────────────────────────────

func TestXSDGroups(t *testing.T) {
	grp := func(seq string) string {
		return xsd(`<xs:group name="G">` +
			`<xs:sequence>` + seq + `</xs:sequence>` +
			`</xs:group>`)
	}

	base := grp(`<xs:element name="a" type="xs:string"/>`)
	addOpt := grp(
		`<xs:element name="a" type="xs:string"/>` +
			`<xs:element name="b" type="xs:string"` +
			` minOccurs="0"/>`,
	)
	removed := xsd(``)

	tests := []struct {
		name    string
		dir     string
		old     string
		new     string
		wantErr bool
	}{
		{
			"add optional to group backward (OK)",
			"backward", base, addOpt, false,
		},
		{
			"remove group backward (FAIL)",
			"backward", base, removed, true,
		},
		{
			"remove group forward (OK)",
			"forward", base, removed, false,
		},
		{
			"add element to group forward (FAIL)",
			"forward", base, addOpt, true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runXSDCompat(t, tc.dir, tc.old, tc.new, tc.wantErr)
		})
	}
}

func TestXSDAttributeGroups(t *testing.T) {
	ag := func(attrs string) string {
		return xsd(`<xs:attributeGroup name="AG">` +
			attrs + `</xs:attributeGroup>`)
	}

	base := ag(
		`<xs:attribute name="id" type="xs:string"` +
			` use="required"/>`,
	)
	addOpt := ag(
		`<xs:attribute name="id" type="xs:string"` +
			` use="required"/>` +
			`<xs:attribute name="tag" type="xs:string"` +
			` use="optional"/>`,
	)
	removed := xsd(``)
	removeAttr := ag(``)

	tests := []struct {
		name    string
		dir     string
		old     string
		new     string
		wantErr bool
	}{
		{
			"add optional attr to group backward (OK)",
			"backward", base, addOpt, false,
		},
		{
			"remove attributeGroup backward (FAIL)",
			"backward", base, removed, true,
		},
		{
			"remove attr from group backward (FAIL)",
			"backward", base, removeAttr, true,
		},
		{
			"remove attributeGroup forward (OK)",
			"forward", base, removed, false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runXSDCompat(t, tc.dir, tc.old, tc.new, tc.wantErr)
		})
	}
}

// ── xs:all and xs:choice tests ────────────────────────────────────

func TestXSDAllAndChoice(t *testing.T) {
	withAll := func(elems string) string {
		return xsd(`<xs:complexType name="T">` +
			`<xs:all>` + elems + `</xs:all>` +
			`</xs:complexType>`)
	}
	withChoice := func(elems string) string {
		return xsd(`<xs:complexType name="T">` +
			`<xs:choice>` + elems + `</xs:choice>` +
			`</xs:complexType>`)
	}

	allBase := withAll(
		`<xs:element name="a" type="xs:string"/>` +
			`<xs:element name="b" type="xs:int"/>`,
	)
	allAddOpt := withAll(
		`<xs:element name="a" type="xs:string"/>` +
			`<xs:element name="b" type="xs:int"/>` +
			`<xs:element name="c" type="xs:string"` +
			` minOccurs="0"/>`,
	)
	allRemove := withAll(
		`<xs:element name="a" type="xs:string"/>`,
	)

	choiceBase := withChoice(
		`<xs:element name="a" type="xs:string"/>` +
			`<xs:element name="b" type="xs:int"/>`,
	)
	choiceAdd := withChoice(
		`<xs:element name="a" type="xs:string"/>` +
			`<xs:element name="b" type="xs:int"/>` +
			`<xs:element name="c" type="xs:string"/>`,
	)
	choiceRemove := withChoice(
		`<xs:element name="a" type="xs:string"/>`,
	)

	tests := []struct {
		name    string
		dir     string
		old     string
		new     string
		wantErr bool
	}{
		// xs:all
		{
			"all add optional backward (OK)",
			"backward", allBase, allAddOpt, false,
		},
		{
			"all remove element backward (FAIL)",
			"backward", allBase, allRemove, true,
		},
		{
			"all add element forward (FAIL)",
			"forward", allBase, allAddOpt, true,
		},
		{
			"all remove element forward (OK for optional)",
			"forward", allAddOpt, allBase, false,
		},
		// xs:choice
		{
			"choice add option backward (OK)",
			"backward", choiceBase, choiceAdd, false,
		},
		{
			"choice remove option backward (FAIL)",
			"backward", choiceBase, choiceRemove, true,
		},
		{
			"choice add option forward (FAIL)",
			"forward", choiceBase, choiceAdd, true,
		},
		{
			"choice remove option forward (OK)",
			"forward", choiceBase, choiceRemove, false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runXSDCompat(t, tc.dir, tc.old, tc.new, tc.wantErr)
		})
	}
}

// ── Inline complexType / simpleType in element tests ─────────────

func TestXSDInlineTypes(t *testing.T) {
	withInlineCT := func(seq string) string {
		return xsd(`<xs:element name="root">` +
			`<xs:complexType><xs:sequence>` +
			seq +
			`</xs:sequence></xs:complexType>` +
			`</xs:element>`)
	}

	base := withInlineCT(
		`<xs:element name="a" type="xs:string"/>`)
	addOpt := withInlineCT(
		`<xs:element name="a" type="xs:string"/>` +
			`<xs:element name="b" type="xs:string"` +
			` minOccurs="0"/>`)
	addReq := withInlineCT(
		`<xs:element name="a" type="xs:string"/>` +
			`<xs:element name="b" type="xs:string"/>`)

	tests := []struct {
		name    string
		dir     string
		old     string
		new     string
		wantErr bool
	}{
		{
			"inline CT add optional backward (OK)",
			"backward", base, addOpt, false,
		},
		{
			"inline CT add required backward (FAIL)",
			"backward", base, addReq, true,
		},
		{
			"inline CT add elem forward (FAIL)",
			"forward", base, addOpt, true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runXSDCompat(t, tc.dir, tc.old, tc.new, tc.wantErr)
		})
	}
}

// ── simpleType list and union ─────────────────────────────────────

func TestXSDSimpleTypeListAndUnion(t *testing.T) {
	list := func(item string) string {
		return xsd(`<xs:simpleType name="S">` +
			`<xs:list itemType="xs:` + item + `"/>` +
			`</xs:simpleType>`)
	}
	union := func(members string) string {
		return xsd(`<xs:simpleType name="S">` +
			`<xs:union memberTypes="` + members + `"/>` +
			`</xs:simpleType>`)
	}

	tests := []struct {
		name    string
		dir     string
		old     string
		new     string
		wantErr bool
	}{
		{
			"list same itemType backward (OK)",
			"backward", list("string"), list("string"), false,
		},
		{
			"list change itemType backward (FAIL)",
			"backward", list("string"), list("integer"), true,
		},
		{
			"union same members backward (OK)",
			"backward",
			union("xs:string xs:integer"),
			union("xs:string xs:integer"),
			false,
		},
		{
			"union add member backward (OK)",
			"backward",
			union("xs:string"),
			union("xs:string xs:integer"),
			false,
		},
		{
			"union remove member backward (FAIL)",
			"backward",
			union("xs:string xs:integer"),
			union("xs:string"),
			true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runXSDCompat(t, tc.dir, tc.old, tc.new, tc.wantErr)
		})
	}
}

// ── Nested compositor tests ───────────────────────────────────────

func TestXSDNestedCompositors(t *testing.T) {
	withNested := func(inner string) string {
		return xsd(`<xs:complexType name="T">` +
			`<xs:sequence>` +
			`<xs:element name="outer" type="xs:string"/>` +
			`<xs:choice>` + inner + `</xs:choice>` +
			`</xs:sequence></xs:complexType>`)
	}

	base := withNested(
		`<xs:element name="a" type="xs:string"/>` +
			`<xs:element name="b" type="xs:int"/>`,
	)
	addChoice := withNested(
		`<xs:element name="a" type="xs:string"/>` +
			`<xs:element name="b" type="xs:int"/>` +
			`<xs:element name="c" type="xs:string"/>`,
	)
	removeChoice := withNested(
		`<xs:element name="a" type="xs:string"/>`,
	)

	tests := []struct {
		name    string
		dir     string
		old     string
		new     string
		wantErr bool
	}{
		{
			"nested choice add option backward (OK)",
			"backward", base, addChoice, false,
		},
		{
			"nested choice remove option backward (FAIL)",
			"backward", base, removeChoice, true,
		},
		{
			"nested choice add forward (FAIL)",
			"forward", base, addChoice, true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runXSDCompat(t, tc.dir, tc.old, tc.new, tc.wantErr)
		})
	}
}

// ── nillable tests ────────────────────────────────────────────────

func TestXSDNillable(t *testing.T) {
	elem := func(nillable string) string {
		return xsd(`<xs:element name="e" type="xs:string"` +
			nillable + `/>`)
	}

	tests := []struct {
		name    string
		dir     string
		old     string
		new     string
		wantErr bool
	}{
		{
			"nillable true→false backward (FAIL)",
			"backward",
			elem(` nillable="true"`),
			elem(``),
			true,
		},
		{
			"nillable false→true backward (OK)",
			"backward",
			elem(``),
			elem(` nillable="true"`),
			false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			runXSDCompat(t, tc.dir, tc.old, tc.new, tc.wantErr)
		})
	}
}
