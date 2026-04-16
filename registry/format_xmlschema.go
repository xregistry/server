// Package registry - XML Schema (XSD) format compatibility checker.
//
// IsValid verifies that a version's document is a syntactically
// valid XML Schema document (XSD). The root element must be
// <schema> in the http://www.w3.org/2001/XMLSchema namespace.
//
// IsCompatible checks whether two XML Schema versions are compatible
// in the given direction. The rules follow the closed-world
// assumption standard for schema registries (producers only emit
// elements defined in their schema):
//
//	"backward" — consumers using the NEW schema can read messages
//	             produced with the OLD schema.
//	             Permitted changes to the schema:
//	               • Add a new top-level element, complexType,
//	                 simpleType, group, or attributeGroup.
//	               • Add an optional element (minOccurs=0) to an
//	                 existing sequence / all / choice.
//	               • Add an optional attribute (use="optional")
//	                 to an existing complexType.
//	               • Add enumeration values to a simpleType
//	                 restriction.
//	               • Widen occurrence constraints (lower minOccurs,
//	                 raise maxOccurs).
//	               • Change attribute use from "required" to
//	                 "optional".
//	             Forbidden changes:
//	               • Remove any top-level declaration.
//	               • Remove a sequence / all / choice element.
//	               • Add a required element (minOccurs≥1) or
//	                 required attribute (use="required").
//	               • Change an element or attribute type.
//	               • Remove an enumeration value from a simpleType.
//	               • Tighten occurrence constraints.
//	             Implemented as: old ⊆ new
//	             (checkXSDBackwardCompat(writer=old, reader=new))
//
//	"forward"  — consumers using the OLD schema can read messages
//	             produced with the NEW schema.
//	             Permitted changes to the schema:
//	               • Remove a top-level element, complexType,
//	                 simpleType, group, or attributeGroup.
//	               • Remove an optional element from a sequence /
//	                 all / choice.
//	               • Remove an optional attribute from a complexType.
//	               • Remove enumeration values.
//	             Forbidden changes:
//	               • Add any element to a sequence / all / choice —
//	                 old consumers may reject the unknown element.
//	               • Add any attribute — old consumers may reject it.
//	               • Change an element or attribute type.
//	             Implemented as: new ⊆ old
//	             (checkXSDBackwardCompat(writer=new, reader=old))
//	             (forward compat = backward compat with args swapped)
//
// Compatibility checks – status per XSD construct:
//
// Top-level declarations
//   - [supported]     element (removal detected; name, type, and
//     inline type body compared)
//   - [supported]     complexType (removal detected; content model
//     and attributes compared recursively)
//   - [supported]     simpleType (removal detected; restriction
//     facets compared individually)
//   - [supported]     group (removal detected; content model
//     compared recursively)
//   - [supported]     attributeGroup (removal detected; attribute
//     list compared)
//   - [not supported] top-level attribute declarations
//   - [not supported] notation
//
// Element / occurrence keywords
//   - [supported]     name (declaration key)
//   - [supported]     type attribute (type change detected)
//   - [supported]     minOccurs (tightening detected)
//   - [supported]     maxOccurs (tightening detected)
//   - [supported]     nillable (true→false detected)
//   - [not supported] default / fixed (not individually tracked)
//   - [not supported] substitutionGroup / block / final / abstract
//
// complexType keywords
//   - [supported]     sequence / all / choice — element particles
//     compared individually by name, type, and occurrence
//   - [supported]     xs:any — namespace / processContents tracked
//   - [supported]     xs:group ref — ref and occurrence tracked
//   - [supported]     nested sequence / all / choice (recursive)
//   - [supported]     attribute (name, type, use tracked)
//   - [supported]     attributeGroup ref (ref tracked)
//   - [supported]     anyAttribute (namespace / processContents)
//   - [supported]     complexContent extension / restriction
//     (base type and content compared)
//   - [supported]     simpleContent extension / restriction
//     (base type compared)
//   - [supported]     mixed flag (true→false detected)
//   - [not supported] abstract / block / final on complexType
//
// simpleType keywords
//   - [supported]     restriction base
//   - [supported]     enumeration (add/remove detected)
//   - [supported]     pattern (change detected)
//   - [supported]     minLength / maxLength (tightening detected)
//   - [supported]     length (change detected)
//   - [supported]     minInclusive / maxInclusive /
//     minExclusive / maxExclusive (tightening detected)
//   - [supported]     totalDigits / fractionDigits (tightening)
//   - [supported]     whiteSpace (change detected)
//   - [supported]     list itemType
//   - [supported]     union memberTypes
//   - [not supported] xs:annotation / xs:documentation (ignored)
//
// Known limitations / not yet implemented
//   - xs:include / xs:import / xs:redefine / xs:override are not
//     resolved; referenced external schemas are not fetched.
//   - Cross-reference resolution: when an element's type="Foo"
//     refers to a named complexType, only the name is compared —
//     the bodies of the two Foo declarations are NOT re-checked
//     in-context (they are checked as top-level declarations).
//   - Sequence ordering: particles must appear in the same relative
//     order in writer and reader sequences; re-ordering existing
//     required elements is detected as incompatible.
//   - Target-namespace changes are not detected.
//   - xs:key / xs:keyref / xs:unique identity constraints
//     are not analysed.

package registry

import (
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	. "github.com/xregistry/server/common"
)

const XMLSCHEMA_FORMAT = "xmlschema.*"

const xmlSchemaNS = "http://www.w3.org/2001/XMLSchema"

func init() {
	RegisterFormat(XMLSCHEMA_FORMAT, FormatXMLSchema{})
}

// FormatXMLSchema implements the Format interface for XML Schema.
type FormatXMLSchema struct{}

func (fx FormatXMLSchema) IsValid(
	ver *Version,
) (string, *XRError) {
	format := ver.GetAsString("format")
	if ok, _ := regexp.MatchString(
		"(?i)"+XMLSCHEMA_FORMAT, format,
	); !ok {
		return "true", NewXRError("bad_request", ver.XID,
			"error_detail="+
				fmt.Sprintf(
					`Version %q has a "format" value of %q,`+
						` was expecting %q`,
					ver.XID, format, XMLSCHEMA_FORMAT))
	}

	if ver.Resource.ResourceModel.GetHasDocument() == false {
		return "true", NewXRError("format_violation", ver.XID,
			"format="+format).
			SetDetailf(
				`The Resource (%s) for Version %q does not`+
					` have "hasdocument" in its resource model`+
					` set to "true", and an empty/missing`+
					` document is not compliant.`,
				ver.Resource.XID, ver.XID)
	}

	if resURL := ver.Get(
		ver.Resource.Singular + "url",
	); !IsNil(resURL) {
		return "false, data stored externally",
			NewXRError("format_external", ver.XID)
	}

	buf := []byte(nil)
	if bufAny := ver.Get(
		ver.Resource.Singular,
	); !IsNil(bufAny) {
		buf = bufAny.([]byte)
	}

	if len(buf) == 0 {
		return "true", NewXRError("format_violation", ver.XID,
			"format="+ver.GetAsString("format")).
			SetDetailf(
				"Version %q is empty and therefore not a "+
					"valid xml schema file.", ver.XID)
	}

	if err := IsValidXMLSchema(buf); err != nil {
		return "true", NewXRError("format_violation", ver.XID,
			"format="+ver.GetAsString("format")).
			SetDetailf(
				"Version %q is not a valid xml schema"+
					" file: %s.", ver.XID, err)
	}

	return "true", nil
}

func (fx FormatXMLSchema) IsCompatible(
	direction string,
	oldVersion *Version,
	newVersion *Version,
) (string, *XRError) {
	reason, xErr := fx.IsValid(oldVersion)
	if xErr != nil {
		return reason, xErr
	}
	reason, xErr = fx.IsValid(newVersion)
	if xErr != nil {
		return reason, xErr
	}

	oldBuf := []byte(nil)
	newBuf := []byte(nil)
	if b := oldVersion.Get(
		oldVersion.Resource.Singular,
	); !IsNil(b) {
		oldBuf = b.([]byte)
	}
	if b := newVersion.Get(
		newVersion.Resource.Singular,
	); !IsNil(b) {
		newBuf = b.([]byte)
	}

	if err := checkXSDCompat(direction, oldBuf, newBuf); err != nil {
		compat := newVersion.
			Resource.
			MustFindMeta(false, FOR_READ).
			GetAsString("compatibility")
		return "true", NewXRError(
			"compatibility_violation", newVersion.XID,
			"compat="+compat).
			SetDetailf(
				"Version %q isn't %q compatible with %q: %s",
				newVersion.XID, compat, oldVersion.XID,
				err.Error())
	}

	return "true", nil
}

// IsValidXMLSchema returns nil when buf is a syntactically valid
// XML Schema document, or an error describing the problem.
func IsValidXMLSchema(buf []byte) error {
	raw, err := parseXSDRaw(buf)
	if err != nil {
		return err
	}
	if raw.XMLName.Local == "" {
		return fmt.Errorf("missing schema root element")
	}
	if raw.XMLName.Local != "schema" {
		return fmt.Errorf("root element must be <schema>")
	}
	ns := strings.TrimSpace(raw.XMLName.Space)
	if ns != "" && ns != xmlSchemaNS {
		return fmt.Errorf(
			"unsupported schema namespace %q", ns)
	}
	return nil
}

// ── Raw XML unmarshaling structs ──────────────────────────────────

// xsdSchemaRaw is the top-level raw unmarshaling target.
type xsdSchemaRaw struct {
	XMLName         xml.Name        `xml:"schema"`
	TargetNamespace string          `xml:"targetNamespace,attr"`
	Elements        []xsdElemRaw    `xml:"element"`
	ComplexTypes    []xsdCTRaw      `xml:"complexType"`
	SimpleTypes     []xsdSTRaw      `xml:"simpleType"`
	Groups          []xsdGroupRaw   `xml:"group"`
	AttrGroups      []xsdAttrGrpRaw `xml:"attributeGroup"`
}

// xsdElemRaw is an xs:element at top-level or inside a compositor.
// Inner captures any inline type definition.
type xsdElemRaw struct {
	Name      string `xml:"name,attr"`
	Ref       string `xml:"ref,attr"`
	TypeRef   string `xml:"type,attr"`
	MinOccurs string `xml:"minOccurs,attr"`
	MaxOccurs string `xml:"maxOccurs,attr"`
	Nillable  string `xml:"nillable,attr"`
	Default   string `xml:"default,attr"`
	Fixed     string `xml:"fixed,attr"`
	Inner     string `xml:",innerxml"`
}

// xsdCTRaw is xs:complexType.
// Direct xs:attribute / xs:attributeGroup / xs:anyAttribute
// children are parsed by encoding/xml. The content model
// (sequence / all / choice / complexContent / simpleContent)
// lives in Inner and is parsed by parseCTInner.
type xsdCTRaw struct {
	Name       string       `xml:"name,attr"`
	Mixed      string       `xml:"mixed,attr"`
	Attributes []xsdAttrRaw `xml:"attribute"`
	AttrGroups []xsdAGRef   `xml:"attributeGroup"`
	AnyAttr    *xsdAnyAttr  `xml:"anyAttribute"`
	Inner      string       `xml:",innerxml"`
}

// xsdSTRaw is xs:simpleType.
type xsdSTRaw struct {
	Name  string `xml:"name,attr"`
	Inner string `xml:",innerxml"`
}

// xsdGroupRaw is xs:group (named model group).
type xsdGroupRaw struct {
	Name  string `xml:"name,attr"`
	Inner string `xml:",innerxml"`
}

// xsdAttrGrpRaw is xs:attributeGroup.
type xsdAttrGrpRaw struct {
	Name       string       `xml:"name,attr"`
	Attributes []xsdAttrRaw `xml:"attribute"`
	AttrGroups []xsdAGRef   `xml:"attributeGroup"`
}

// xsdAttrRaw is xs:attribute.
type xsdAttrRaw struct {
	Name    string `xml:"name,attr"`
	Ref     string `xml:"ref,attr"`
	TypeRef string `xml:"type,attr"`
	Use     string `xml:"use,attr"`
	Default string `xml:"default,attr"`
	Fixed   string `xml:"fixed,attr"`
}

// xsdAGRef is xs:attributeGroup ref="..".
type xsdAGRef struct {
	Ref string `xml:"ref,attr"`
}

// xsdAnyAttr is xs:anyAttribute.
type xsdAnyAttr struct {
	Namespace       string `xml:"namespace,attr"`
	ProcessContents string `xml:"processContents,attr"`
}

// ── Parsed (rich) model ───────────────────────────────────────────

// xsdSchema is the fully-parsed schema model.
type xsdSchema struct {
	TargetNamespace string
	Elements        map[string]*xsdElement
	ComplexTypes    map[string]*xsdComplexType
	SimpleTypes     map[string]*xsdSimpleType
	Groups          map[string]*xsdGroup
	AttrGroups      map[string]*xsdAttrGroup
}

// xsdOccurs holds minOccurs / maxOccurs; Max == -1 means unbounded.
type xsdOccurs struct {
	Min int
	Max int
}

func defaultOccurs() xsdOccurs { return xsdOccurs{1, 1} }

// xsdElement is a parsed xs:element declaration.
type xsdElement struct {
	Name     string
	Ref      string
	TypeRef  string
	Occurs   xsdOccurs
	Nillable bool
	InlineCT *xsdComplexType
	InlineST *xsdSimpleType
}

// xsdComplexType is a parsed xs:complexType.
type xsdComplexType struct {
	Name      string
	Mixed     bool
	DerivKind string // "", "extension", "restriction"
	BaseType  string
	Content   *xsdCompositor
	Attrs     []*xsdAttrUse
	AnyAttr   *xsdAnyAttr
}

// xsdCompositor is a parsed xs:sequence, xs:all, or xs:choice.
type xsdCompositor struct {
	Kind      string // "sequence", "all", "choice"
	Occurs    xsdOccurs
	Particles []xsdParticle
}

// xsdParticle is one entry in a compositor's particle list.
type xsdParticle interface{ particleKind() string }

type xsdElemParticle struct{ Elem *xsdElement }

func (xsdElemParticle) particleKind() string { return "element" }

type xsdCompParticle struct{ Comp *xsdCompositor }

func (xsdCompParticle) particleKind() string { return "compositor" }

type xsdGroupRefParticle struct {
	Ref    string
	Occurs xsdOccurs
}

func (xsdGroupRefParticle) particleKind() string {
	return "groupRef"
}

type xsdAnyParticle struct {
	Namespace       string
	ProcessContents string
	Occurs          xsdOccurs
}

func (xsdAnyParticle) particleKind() string { return "any" }

// xsdAttrUse is a parsed xs:attribute within a complexType or
// attributeGroup.
type xsdAttrUse struct {
	Name     string
	TypeRef  string
	Use      string // "required", "optional", or "prohibited"
	Default  string
	Fixed    string
	GroupRef string // non-empty when from xs:attributeGroup ref
}

// xsdSimpleType is a parsed xs:simpleType.
type xsdSimpleType struct {
	Name           string
	DerivKind      string // "restriction", "list", "union"
	BaseType       string
	Enumerations   []string
	Patterns       []string
	MinInclusive   string
	MaxInclusive   string
	MinExclusive   string
	MaxExclusive   string
	Length         *int
	MinLength      *int
	MaxLength      *int
	TotalDigits    *int
	FractionDigits *int
	WhiteSpace     string
	ListItemType   string
	UnionMembers   []string
}

// xsdGroup is a parsed xs:group (named model group).
type xsdGroup struct {
	Name    string
	Content *xsdCompositor
}

// xsdAttrGroup is a parsed xs:attributeGroup.
type xsdAttrGroup struct {
	Name  string
	Attrs []*xsdAttrUse
}

// ── Parsing: raw structs → parsed model ───────────────────────────

func parseXSDRaw(buf []byte) (*xsdSchemaRaw, error) {
	var raw xsdSchemaRaw
	if err := xml.Unmarshal(buf, &raw); err != nil {
		return nil, err
	}
	return &raw, nil
}

// parseXSD converts raw bytes into the rich parsed model.
func parseXSD(buf []byte) (*xsdSchema, error) {
	if err := IsValidXMLSchema(buf); err != nil {
		return nil, err
	}
	raw, err := parseXSDRaw(buf)
	if err != nil {
		return nil, err
	}

	s := &xsdSchema{
		TargetNamespace: raw.TargetNamespace,
		Elements:        map[string]*xsdElement{},
		ComplexTypes:    map[string]*xsdComplexType{},
		SimpleTypes:     map[string]*xsdSimpleType{},
		Groups:          map[string]*xsdGroup{},
		AttrGroups:      map[string]*xsdAttrGroup{},
	}

	for i := range raw.Elements {
		e := &raw.Elements[i]
		if e.Name == "" {
			continue
		}
		pe, err := parseElemRaw(e)
		if err != nil {
			return nil, fmt.Errorf(
				"element %q: %w", e.Name, err)
		}
		s.Elements[e.Name] = pe
	}

	for i := range raw.ComplexTypes {
		ct := &raw.ComplexTypes[i]
		if ct.Name == "" {
			continue
		}
		pct, err := parseCTRaw(ct)
		if err != nil {
			return nil, fmt.Errorf(
				"complexType %q: %w", ct.Name, err)
		}
		s.ComplexTypes[ct.Name] = pct
	}

	for i := range raw.SimpleTypes {
		st := &raw.SimpleTypes[i]
		if st.Name == "" {
			continue
		}
		pst, err := parseSTFromInner(st.Name, st.Inner)
		if err != nil {
			return nil, fmt.Errorf(
				"simpleType %q: %w", st.Name, err)
		}
		s.SimpleTypes[st.Name] = pst
	}

	for i := range raw.Groups {
		g := &raw.Groups[i]
		if g.Name == "" {
			continue
		}
		pg, err := parseGroupRaw(g)
		if err != nil {
			return nil, fmt.Errorf(
				"group %q: %w", g.Name, err)
		}
		s.Groups[g.Name] = pg
	}

	for i := range raw.AttrGroups {
		ag := &raw.AttrGroups[i]
		if ag.Name == "" {
			continue
		}
		s.AttrGroups[ag.Name] = parseAttrGrpRaw(ag)
	}

	return s, nil
}

// parseElemRaw converts an xsdElemRaw into an xsdElement.
func parseElemRaw(e *xsdElemRaw) (*xsdElement, error) {
	pe := &xsdElement{
		Name:     e.Name,
		Ref:      xsdLocalName(e.Ref),
		TypeRef:  xsdLocalName(e.TypeRef),
		Occurs:   parseOccurs(e.MinOccurs, e.MaxOccurs),
		Nillable: strings.EqualFold(e.Nillable, "true"),
	}
	if e.Inner != "" {
		ct, st, err := parseInlineType(e.Inner)
		if err != nil {
			return nil, err
		}
		pe.InlineCT = ct
		pe.InlineST = st
	}
	return pe, nil
}

// parseCTRaw converts an xsdCTRaw into an xsdComplexType.
func parseCTRaw(ct *xsdCTRaw) (*xsdComplexType, error) {
	pct := &xsdComplexType{
		Name:    ct.Name,
		Mixed:   strings.EqualFold(ct.Mixed, "true"),
		AnyAttr: ct.AnyAttr,
	}
	for _, ar := range ct.Attributes {
		pct.Attrs = append(pct.Attrs, parseAttrRaw(&ar))
	}
	for _, ag := range ct.AttrGroups {
		pct.Attrs = append(pct.Attrs, &xsdAttrUse{
			GroupRef: xsdLocalName(ag.Ref),
			Use:      "optional",
		})
	}
	if ct.Inner != "" {
		comp, kind, base, err := parseCTInner(ct.Inner)
		if err != nil {
			return nil, err
		}
		pct.Content = comp
		pct.DerivKind = kind
		pct.BaseType = xsdLocalName(base)
	}
	return pct, nil
}

// parseAttrRaw converts an xsdAttrRaw into an xsdAttrUse.
func parseAttrRaw(a *xsdAttrRaw) *xsdAttrUse {
	use := strings.ToLower(a.Use)
	if use == "" {
		use = "optional"
	}
	return &xsdAttrUse{
		Name:    xsdLocalName(a.Name),
		TypeRef: xsdLocalName(a.TypeRef),
		Use:     use,
		Default: a.Default,
		Fixed:   a.Fixed,
	}
}

// parseGroupRaw converts an xsdGroupRaw into an xsdGroup.
func parseGroupRaw(g *xsdGroupRaw) (*xsdGroup, error) {
	pg := &xsdGroup{Name: g.Name}
	if g.Inner != "" {
		comp, err := parseParticlesFromInner(g.Inner)
		if err != nil {
			return nil, err
		}
		pg.Content = comp
	}
	return pg, nil
}

// parseAttrGrpRaw converts an xsdAttrGrpRaw into an xsdAttrGroup.
func parseAttrGrpRaw(ag *xsdAttrGrpRaw) *xsdAttrGroup {
	pag := &xsdAttrGroup{Name: ag.Name}
	for _, a := range ag.Attributes {
		pag.Attrs = append(pag.Attrs, parseAttrRaw(&a))
	}
	for _, ref := range ag.AttrGroups {
		pag.Attrs = append(pag.Attrs, &xsdAttrUse{
			GroupRef: xsdLocalName(ref.Ref),
			Use:      "optional",
		})
	}
	return pag
}

// ── Token-based inner-XML parsers ─────────────────────────────────

// xsdWrapInner wraps raw inner XML in a temporary root element so
// that xml.NewDecoder can tokenize it with namespace context.
func xsdWrapInner(inner string) *xml.Decoder {
	wrapped := `<_root xmlns:xs="` + xmlSchemaNS + `">` +
		inner + `</_root>`
	return xml.NewDecoder(strings.NewReader(wrapped))
}

// xsdSkipToFirstStart advances the decoder past the first start
// element (the artificial root created by xsdWrapInner).
func xsdSkipToFirstStart(dec *xml.Decoder) error {
	for {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		if _, ok := tok.(xml.StartElement); ok {
			return nil
		}
	}
}

// parseInlineType parses the inner XML of an xs:element to extract
// an inline complexType or simpleType.
func parseInlineType(
	inner string,
) (*xsdComplexType, *xsdSimpleType, error) {
	dec := xsdWrapInner(inner)
	if err := xsdSkipToFirstStart(dec); err != nil {
		return nil, nil, nil
	}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil, nil, nil
		}
		if err != nil {
			return nil, nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "complexType":
			var raw xsdCTRaw
			if err := dec.DecodeElement(&raw, &se); err != nil {
				return nil, nil, err
			}
			pct, err := parseCTRaw(&raw)
			if err != nil {
				return nil, nil, err
			}
			return pct, nil, nil
		case "simpleType":
			var raw xsdSTRaw
			if err := dec.DecodeElement(&raw, &se); err != nil {
				return nil, nil, err
			}
			pst, err := parseSTFromInner(raw.Name, raw.Inner)
			if err != nil {
				return nil, nil, err
			}
			return nil, pst, nil
		default:
			if err := dec.Skip(); err != nil {
				return nil, nil, err
			}
		}
	}
}

// parseCTInner parses the inner XML of an xs:complexType to extract
// the content model, derivation kind, and base type.
// Returns (compositor, derivKind, baseType, error).
func parseCTInner(
	inner string,
) (*xsdCompositor, string, string, error) {
	dec := xsdWrapInner(inner)
	if err := xsdSkipToFirstStart(dec); err != nil {
		return nil, "", "", nil
	}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil, "", "", nil
		}
		if err != nil {
			return nil, "", "", err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "sequence", "all", "choice":
			comp, err := parseCompositorFromSE(dec, se)
			if err != nil {
				return nil, "", "", err
			}
			return comp, "", "", nil
		case "complexContent":
			kind, base, comp, err :=
				parseComplexContentSE(dec, se)
			if err != nil {
				return nil, "", "", err
			}
			return comp, kind, base, nil
		case "simpleContent":
			kind, base, err :=
				parseSimpleContentSE(dec, se)
			if err != nil {
				return nil, "", "", err
			}
			return nil, kind, base, nil
		default:
			if err := dec.Skip(); err != nil {
				return nil, "", "", err
			}
		}
	}
}

// parseComplexContentSE parses xs:complexContent and returns
// (derivKind, baseType, compositor, error).
func parseComplexContentSE(
	dec *xml.Decoder,
	se xml.StartElement,
) (string, string, *xsdCompositor, error) {
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", "", nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "extension", "restriction":
				kind := t.Name.Local
				base := xsdAttrVal(t, "base")
				comp, err := parseDerivContentSE(dec, t)
				if err != nil {
					return "", "", nil, err
				}
				xsdConsumeToEnd(
					dec, se.Name.Local)
				return kind, base, comp, nil
			default:
				if err := dec.Skip(); err != nil {
					return "", "", nil, err
				}
			}
		case xml.EndElement:
			return "", "", nil, nil
		}
	}
}

// parseSimpleContentSE parses xs:simpleContent and returns
// (derivKind, baseType, error).
func parseSimpleContentSE(
	dec *xml.Decoder,
	se xml.StartElement,
) (string, string, error) {
	for {
		tok, err := dec.Token()
		if err != nil {
			return "", "", err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "extension", "restriction":
				kind := t.Name.Local
				base := xsdAttrVal(t, "base")
				if err := dec.Skip(); err != nil {
					return "", "", err
				}
				xsdConsumeToEnd(
					dec, se.Name.Local)
				return kind, base, nil
			default:
				if err := dec.Skip(); err != nil {
					return "", "", err
				}
			}
		case xml.EndElement:
			return "", "", nil
		}
	}
}

// parseDerivContentSE parses the compositor inside
// xs:complexContent/xs:extension or xs:restriction.
func parseDerivContentSE(
	dec *xml.Decoder,
	se xml.StartElement,
) (*xsdCompositor, error) {
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "sequence", "all", "choice":
				comp, err := parseCompositorFromSE(dec, t)
				if err != nil {
					return nil, err
				}
				xsdConsumeToEnd(
					dec, se.Name.Local)
				return comp, nil
			default:
				if err := dec.Skip(); err != nil {
					return nil, err
				}
			}
		case xml.EndElement:
			return nil, nil
		}
	}
}

// parseParticlesFromInner parses the inner XML of a group and
// returns the first compositor found.
func parseParticlesFromInner(
	inner string,
) (*xsdCompositor, error) {
	dec := xsdWrapInner(inner)
	if err := xsdSkipToFirstStart(dec); err != nil {
		return nil, nil
	}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "sequence", "all", "choice":
			return parseCompositorFromSE(dec, se)
		default:
			if err := dec.Skip(); err != nil {
				return nil, err
			}
		}
	}
}

// parseCompositorFromSE recursively parses a compositor
// (xs:sequence, xs:all, xs:choice) starting from the given
// already-consumed start element.
func parseCompositorFromSE(
	dec *xml.Decoder,
	se xml.StartElement,
) (*xsdCompositor, error) {
	comp := &xsdCompositor{
		Kind:   se.Name.Local,
		Occurs: parseOccursFromSE(se),
	}
	for {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			p, err := parseParticle(dec, t)
			if err != nil {
				return nil, err
			}
			if p != nil {
				comp.Particles = append(
					comp.Particles, p)
			}
		case xml.EndElement:
			return comp, nil
		}
	}
}

// parseParticle parses one particle inside a compositor.
func parseParticle(
	dec *xml.Decoder,
	se xml.StartElement,
) (xsdParticle, error) {
	switch se.Name.Local {
	case "element":
		var raw xsdElemRaw
		if err := dec.DecodeElement(&raw, &se); err != nil {
			return nil, err
		}
		elem, err := parseElemRaw(&raw)
		if err != nil {
			return nil, err
		}
		return xsdElemParticle{Elem: elem}, nil

	case "sequence", "all", "choice":
		comp, err := parseCompositorFromSE(dec, se)
		if err != nil {
			return nil, err
		}
		return xsdCompParticle{Comp: comp}, nil

	case "group":
		ref := xsdAttrVal(se, "ref")
		occurs := parseOccursFromSE(se)
		if err := dec.Skip(); err != nil {
			return nil, err
		}
		return xsdGroupRefParticle{
			Ref:    xsdLocalName(ref),
			Occurs: occurs,
		}, nil

	case "any":
		ns := xsdAttrVal(se, "namespace")
		pc := xsdAttrVal(se, "processContents")
		occurs := parseOccursFromSE(se)
		if err := dec.Skip(); err != nil {
			return nil, err
		}
		return xsdAnyParticle{
			Namespace:       ns,
			ProcessContents: pc,
			Occurs:          occurs,
		}, nil

	default:
		if err := dec.Skip(); err != nil {
			return nil, err
		}
		return nil, nil
	}
}

// parseSTFromInner parses the inner XML of an xs:simpleType into
// an xsdSimpleType.
func parseSTFromInner(
	name string, inner string,
) (*xsdSimpleType, error) {
	st := &xsdSimpleType{Name: name}
	if inner == "" {
		return st, nil
	}
	dec := xsdWrapInner(inner)
	if err := xsdSkipToFirstStart(dec); err != nil {
		return st, nil
	}
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			return st, nil
		}
		if err != nil {
			return nil, err
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "restriction":
			st.DerivKind = "restriction"
			st.BaseType = xsdLocalName(
				xsdAttrVal(se, "base"))
			if err := parseRestrictionFacets(
				dec, st,
			); err != nil {
				return nil, err
			}
		case "list":
			st.DerivKind = "list"
			st.ListItemType = xsdLocalName(
				xsdAttrVal(se, "itemType"))
			if err := dec.Skip(); err != nil {
				return nil, err
			}
		case "union":
			st.DerivKind = "union"
			mt := xsdAttrVal(se, "memberTypes")
			for _, m := range strings.Fields(mt) {
				st.UnionMembers = append(
					st.UnionMembers,
					xsdLocalName(m))
			}
			if err := dec.Skip(); err != nil {
				return nil, err
			}
		default:
			if err := dec.Skip(); err != nil {
				return nil, err
			}
		}
	}
}

// parseRestrictionFacets reads the facet children of xs:restriction.
func parseRestrictionFacets(
	dec *xml.Decoder,
	st *xsdSimpleType,
) error {
	for {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			val := xsdAttrVal(t, "value")
			switch t.Name.Local {
			case "enumeration":
				st.Enumerations = append(
					st.Enumerations, val)
			case "pattern":
				st.Patterns = append(st.Patterns, val)
			case "minInclusive":
				st.MinInclusive = val
			case "maxInclusive":
				st.MaxInclusive = val
			case "minExclusive":
				st.MinExclusive = val
			case "maxExclusive":
				st.MaxExclusive = val
			case "length":
				n := xsdParseIntFacet(val)
				st.Length = &n
			case "minLength":
				n := xsdParseIntFacet(val)
				st.MinLength = &n
			case "maxLength":
				n := xsdParseIntFacet(val)
				st.MaxLength = &n
			case "totalDigits":
				n := xsdParseIntFacet(val)
				st.TotalDigits = &n
			case "fractionDigits":
				n := xsdParseIntFacet(val)
				st.FractionDigits = &n
			case "whiteSpace":
				st.WhiteSpace = val
			}
			if err := dec.Skip(); err != nil {
				return err
			}
		case xml.EndElement:
			return nil
		}
	}
}

// ── Compatibility check ───────────────────────────────────────────

// checkXSDCompat is the direction dispatcher.
// "forward": new ⊆ old — implemented by swapping the arguments.
func checkXSDCompat(
	direction string,
	oldBuf, newBuf []byte,
) error {
	oldS, err := parseXSD(oldBuf)
	if err != nil {
		return fmt.Errorf("old schema: %w", err)
	}
	newS, err := parseXSD(newBuf)
	if err != nil {
		return fmt.Errorf("new schema: %w", err)
	}

	writer, reader := oldS, newS
	if strings.EqualFold(direction, "forward") {
		writer, reader = newS, oldS
	}
	return checkXSDBackwardCompat(writer, reader)
}

// checkXSDBackwardCompat checks writer ⊆ reader: every document
// valid against the writer schema is also valid against the reader
// schema.
func checkXSDBackwardCompat(
	writer, reader *xsdSchema,
) error {
	if writer.TargetNamespace != reader.TargetNamespace {
		return fmt.Errorf(
			"target namespace changed from %q to %q",
			writer.TargetNamespace,
			reader.TargetNamespace)
	}

	for name, we := range writer.Elements {
		re, ok := reader.Elements[name]
		if !ok {
			return fmt.Errorf(
				"top-level element %q removed", name)
		}
		if err := checkElemCompat(we, re); err != nil {
			return fmt.Errorf(
				"top-level element %q: %v", name, err)
		}
	}

	for name, wct := range writer.ComplexTypes {
		rct, ok := reader.ComplexTypes[name]
		if !ok {
			return fmt.Errorf(
				"complexType %q removed", name)
		}
		if err := checkCTCompat(wct, rct); err != nil {
			return fmt.Errorf(
				"complexType %q: %v", name, err)
		}
	}

	for name, wst := range writer.SimpleTypes {
		rst, ok := reader.SimpleTypes[name]
		if !ok {
			return fmt.Errorf(
				"simpleType %q removed", name)
		}
		if err := checkSTCompat(wst, rst); err != nil {
			return fmt.Errorf(
				"simpleType %q: %v", name, err)
		}
	}

	for name, wg := range writer.Groups {
		rg, ok := reader.Groups[name]
		if !ok {
			return fmt.Errorf("group %q removed", name)
		}
		if err := checkCompositorCompat(
			wg.Content, rg.Content,
		); err != nil {
			return fmt.Errorf(
				"group %q: %v", name, err)
		}
	}

	for name, wag := range writer.AttrGroups {
		rag, ok := reader.AttrGroups[name]
		if !ok {
			return fmt.Errorf(
				"attributeGroup %q removed", name)
		}
		if err := checkAttrsCompat(
			wag.Attrs, rag.Attrs,
		); err != nil {
			return fmt.Errorf(
				"attributeGroup %q: %v", name, err)
		}
	}

	return nil
}

// checkElemCompat checks that a reader element is backward-
// compatible with the writer element.
func checkElemCompat(w, r *xsdElement) error {
	if w.TypeRef != r.TypeRef {
		return fmt.Errorf(
			"type changed from %q to %q",
			w.TypeRef, r.TypeRef)
	}
	if w.Nillable && !r.Nillable {
		return fmt.Errorf(
			"nillable changed from true to false")
	}
	if err := checkOccursCompat(w.Occurs, r.Occurs); err != nil {
		return err
	}
	if w.InlineCT != nil {
		if r.InlineCT == nil {
			return fmt.Errorf("inline complexType removed")
		}
		if err := checkCTCompat(
			w.InlineCT, r.InlineCT,
		); err != nil {
			return fmt.Errorf(
				"inline complexType: %v", err)
		}
	}
	if w.InlineST != nil {
		if r.InlineST == nil {
			return fmt.Errorf("inline simpleType removed")
		}
		if err := checkSTCompat(
			w.InlineST, r.InlineST,
		); err != nil {
			return fmt.Errorf(
				"inline simpleType: %v", err)
		}
	}
	return nil
}

// checkOccursCompat checks that reader constraints are not tighter
// than writer constraints.
func checkOccursCompat(w, r xsdOccurs) error {
	if r.Min > w.Min {
		return fmt.Errorf(
			"minOccurs increased from %d to %d",
			w.Min, r.Min)
	}
	// -1 = unbounded; unbounded reader is always OK
	if r.Max != -1 && (w.Max == -1 || r.Max < w.Max) {
		return fmt.Errorf(
			"maxOccurs decreased from %s to %d",
			xsdOccursStr(w.Max), r.Max)
	}
	return nil
}

// checkCTCompat checks that a reader complexType is backward-
// compatible with the writer complexType.
func checkCTCompat(w, r *xsdComplexType) error {
	if w.Mixed && !r.Mixed {
		return fmt.Errorf(
			"mixed content changed from true to false")
	}
	if w.DerivKind != r.DerivKind {
		return fmt.Errorf(
			"derivation kind changed from %q to %q",
			w.DerivKind, r.DerivKind)
	}
	if w.BaseType != r.BaseType {
		return fmt.Errorf(
			"base type changed from %q to %q",
			w.BaseType, r.BaseType)
	}
	if err := checkCompositorCompat(
		w.Content, r.Content,
	); err != nil {
		return err
	}
	return checkAttrsCompat(w.Attrs, r.Attrs)
}

// checkCompositorCompat checks that a reader compositor is backward-
// compatible with the writer compositor.
func checkCompositorCompat(
	w, r *xsdCompositor,
) error {
	if w == nil && r == nil {
		return nil
	}
	if w == nil {
		// Reader added a content model; OK only if all optional.
		for _, p := range r.Particles {
			if xsdMinOccurs(p) > 0 {
				return fmt.Errorf(
					"content model added with " +
						"required particles")
			}
		}
		return nil
	}
	if r == nil {
		return fmt.Errorf("content model removed")
	}
	if w.Kind != r.Kind {
		return fmt.Errorf(
			"compositor kind changed from %q to %q",
			w.Kind, r.Kind)
	}
	if err := checkOccursCompat(
		w.Occurs, r.Occurs,
	); err != nil {
		return err
	}

	switch w.Kind {
	case "all":
		return checkAllCompat(
			w.Particles, r.Particles)
	case "choice":
		return checkChoiceCompat(
			w.Particles, r.Particles)
	default: // "sequence"
		return checkSequenceCompat(
			w.Particles, r.Particles)
	}
}

// checkSequenceCompat checks sequence particle compatibility.
// All writer particles must appear in the reader in the same
// relative order. Reader may insert optional particles between them.
func checkSequenceCompat(
	wp, rp []xsdParticle,
) error {
	ri := 0
	for _, w := range wp {
		found := false
		for ri < len(rp) {
			if xsdParticlesMatch(w, rp[ri]) {
				if err := checkParticleCompat(
					w, rp[ri],
				); err != nil {
					return err
				}
				ri++
				found = true
				break
			}
			// Reader has an extra particle before the
			// match; it must be optional.
			if xsdMinOccurs(rp[ri]) > 0 {
				return fmt.Errorf(
					"reader sequence inserts required"+
						" particle %q before writer"+
						" particle %q",
					xsdParticleName(rp[ri]),
					xsdParticleName(w))
			}
			ri++
		}
		if !found {
			return fmt.Errorf(
				"particle %q removed from sequence",
				xsdParticleName(w))
		}
	}
	// Remaining reader-only particles must be optional.
	for _, r := range rp[ri:] {
		if xsdMinOccurs(r) > 0 {
			return fmt.Errorf(
				"reader sequence adds required"+
					" particle %q",
				xsdParticleName(r))
		}
	}
	return nil
}

// checkAllCompat checks xs:all particle compatibility (order-free).
func checkAllCompat(wp, rp []xsdParticle) error {
	rByName := map[string]xsdParticle{}
	for _, p := range rp {
		rByName[xsdParticleName(p)] = p
	}
	for _, w := range wp {
		r, ok := rByName[xsdParticleName(w)]
		if !ok {
			return fmt.Errorf(
				"particle %q removed from all",
				xsdParticleName(w))
		}
		if err := checkParticleCompat(w, r); err != nil {
			return err
		}
	}
	wByName := map[string]bool{}
	for _, p := range wp {
		wByName[xsdParticleName(p)] = true
	}
	for _, r := range rp {
		if !wByName[xsdParticleName(r)] &&
			xsdMinOccurs(r) > 0 {
			return fmt.Errorf(
				"all adds required particle %q",
				xsdParticleName(r))
		}
	}
	return nil
}

// checkChoiceCompat checks xs:choice particle compatibility.
// All writer choices must still be present in the reader.
func checkChoiceCompat(wp, rp []xsdParticle) error {
	rByName := map[string]xsdParticle{}
	for _, p := range rp {
		rByName[xsdParticleName(p)] = p
	}
	for _, w := range wp {
		r, ok := rByName[xsdParticleName(w)]
		if !ok {
			return fmt.Errorf(
				"choice option %q removed",
				xsdParticleName(w))
		}
		if err := checkParticleCompat(w, r); err != nil {
			return err
		}
	}
	return nil
}

// checkParticleCompat checks two matching particles for
// compatibility.
func checkParticleCompat(w, r xsdParticle) error {
	switch wt := w.(type) {
	case xsdElemParticle:
		rt := r.(xsdElemParticle)
		return checkElemCompat(wt.Elem, rt.Elem)
	case xsdCompParticle:
		rt := r.(xsdCompParticle)
		return checkCompositorCompat(wt.Comp, rt.Comp)
	case xsdGroupRefParticle:
		rt := r.(xsdGroupRefParticle)
		if wt.Ref != rt.Ref {
			return fmt.Errorf(
				"group ref changed from %q to %q",
				wt.Ref, rt.Ref)
		}
		return checkOccursCompat(wt.Occurs, rt.Occurs)
	case xsdAnyParticle:
		rt := r.(xsdAnyParticle)
		if wt.Namespace != rt.Namespace ||
			wt.ProcessContents != rt.ProcessContents {
			return fmt.Errorf(
				"xs:any constraints changed")
		}
		return checkOccursCompat(wt.Occurs, rt.Occurs)
	}
	return nil
}

// checkAttrsCompat checks that reader attributes are backward-
// compatible with writer attributes.
func checkAttrsCompat(
	wAttrs, rAttrs []*xsdAttrUse,
) error {
	rByName := map[string]*xsdAttrUse{}
	for _, a := range rAttrs {
		rByName[xsdAttrKey(a)] = a
	}

	for _, w := range wAttrs {
		key := xsdAttrKey(w)
		r, ok := rByName[key]
		if !ok {
			return fmt.Errorf(
				"attribute %q removed", key)
		}
		if w.TypeRef != r.TypeRef {
			return fmt.Errorf(
				"attribute %q type changed"+
					" from %q to %q",
				key, w.TypeRef, r.TypeRef)
		}
		// optional/unset → required is a tightening.
		if w.Use != "required" && r.Use == "required" {
			return fmt.Errorf(
				"attribute %q use changed to required",
				key)
		}
	}

	// Reader adds new attributes.
	wByName := map[string]bool{}
	for _, a := range wAttrs {
		wByName[xsdAttrKey(a)] = true
	}
	for _, r := range rAttrs {
		if !wByName[xsdAttrKey(r)] &&
			r.Use == "required" {
			return fmt.Errorf(
				"attribute %q added as required",
				xsdAttrKey(r))
		}
	}
	return nil
}

// checkSTCompat checks that a reader simpleType is backward-
// compatible with the writer simpleType.
func checkSTCompat(w, r *xsdSimpleType) error {
	if w.DerivKind != r.DerivKind {
		return fmt.Errorf(
			"derivation kind changed from %q to %q",
			w.DerivKind, r.DerivKind)
	}
	if w.BaseType != r.BaseType {
		return fmt.Errorf(
			"base type changed from %q to %q",
			w.BaseType, r.BaseType)
	}

	switch w.DerivKind {
	case "restriction":
		return checkRestrictionCompat(w, r)
	case "list":
		if w.ListItemType != r.ListItemType {
			return fmt.Errorf(
				"list itemType changed from %q to %q",
				w.ListItemType, r.ListItemType)
		}
	case "union":
		return checkUnionCompat(
			w.UnionMembers, r.UnionMembers)
	}
	return nil
}

// checkRestrictionCompat checks restriction facet compatibility.
func checkRestrictionCompat(w, r *xsdSimpleType) error {
	// Enumerations: all writer values must survive in reader.
	if len(w.Enumerations) > 0 {
		rEnums := map[string]bool{}
		for _, e := range r.Enumerations {
			rEnums[e] = true
		}
		for _, e := range w.Enumerations {
			if !rEnums[e] {
				return fmt.Errorf(
					"enumeration value %q removed", e)
			}
		}
	}

	if !xsdStrSliceEq(w.Patterns, r.Patterns) {
		return fmt.Errorf("patterns changed")
	}

	if w.MinInclusive != r.MinInclusive {
		return fmt.Errorf(
			"minInclusive changed from %q to %q",
			w.MinInclusive, r.MinInclusive)
	}
	if w.MaxInclusive != r.MaxInclusive {
		return fmt.Errorf(
			"maxInclusive changed from %q to %q",
			w.MaxInclusive, r.MaxInclusive)
	}
	if w.MinExclusive != r.MinExclusive {
		return fmt.Errorf(
			"minExclusive changed from %q to %q",
			w.MinExclusive, r.MinExclusive)
	}
	if w.MaxExclusive != r.MaxExclusive {
		return fmt.Errorf(
			"maxExclusive changed from %q to %q",
			w.MaxExclusive, r.MaxExclusive)
	}

	if err := xsdCheckLenFacet(
		"length", w.Length, r.Length, false,
	); err != nil {
		return err
	}
	if err := xsdCheckLenFacet(
		"minLength", w.MinLength, r.MinLength, true,
	); err != nil {
		return err
	}
	if err := xsdCheckLenFacet(
		"maxLength", w.MaxLength, r.MaxLength, false,
	); err != nil {
		return err
	}
	if err := xsdCheckLenFacet(
		"totalDigits", w.TotalDigits, r.TotalDigits, false,
	); err != nil {
		return err
	}
	if err := xsdCheckLenFacet(
		"fractionDigits", w.FractionDigits,
		r.FractionDigits, false,
	); err != nil {
		return err
	}
	if w.WhiteSpace != r.WhiteSpace {
		return fmt.Errorf(
			"whiteSpace changed from %q to %q",
			w.WhiteSpace, r.WhiteSpace)
	}
	return nil
}

// xsdCheckLenFacet compares a length-style facet.
// isMin=true: an increase is tightening (incompatible).
// isMin=false: a decrease is tightening (incompatible).
func xsdCheckLenFacet(
	name string, w, r *int, isMin bool,
) error {
	if w == nil || r == nil {
		return nil
	}
	if *w == *r {
		return nil
	}
	if name == "length" {
		return fmt.Errorf(
			"length changed from %d to %d", *w, *r)
	}
	if isMin && *r > *w {
		return fmt.Errorf(
			"%s increased from %d to %d", name, *w, *r)
	}
	if !isMin && *r < *w {
		return fmt.Errorf(
			"%s decreased from %d to %d", name, *w, *r)
	}
	return nil
}

// checkUnionCompat checks union memberTypes compatibility.
func checkUnionCompat(wMembers, rMembers []string) error {
	rSet := map[string]bool{}
	for _, m := range rMembers {
		rSet[m] = true
	}
	for _, m := range wMembers {
		if !rSet[m] {
			return fmt.Errorf(
				"union member type %q removed", m)
		}
	}
	return nil
}

// ── Helper functions ──────────────────────────────────────────────

// parseOccurs converts minOccurs/maxOccurs attribute strings.
// XSD defaults: min=1, max=1.
func parseOccurs(minStr, maxStr string) xsdOccurs {
	o := defaultOccurs()
	if minStr != "" {
		if n, err := strconv.Atoi(minStr); err == nil {
			o.Min = n
		}
	}
	if maxStr != "" {
		if strings.EqualFold(maxStr, "unbounded") {
			o.Max = -1
		} else if n, err := strconv.Atoi(maxStr); err == nil {
			o.Max = n
		}
	}
	return o
}

// parseOccursFromSE extracts occurrence attributes from a start
// element.
func parseOccursFromSE(se xml.StartElement) xsdOccurs {
	return parseOccurs(
		xsdAttrVal(se, "minOccurs"),
		xsdAttrVal(se, "maxOccurs"),
	)
}

// xsdAttrVal returns the value of the named attribute, or "".
func xsdAttrVal(se xml.StartElement, name string) string {
	for _, a := range se.Attr {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

// xsdLocalName strips any namespace prefix (e.g. "xs:string"→
// "string").
func xsdLocalName(s string) string {
	if i := strings.LastIndex(s, ":"); i >= 0 {
		return s[i+1:]
	}
	return s
}

// xsdAttrKey returns the map key for an xsdAttrUse.
func xsdAttrKey(a *xsdAttrUse) string {
	if a.Name != "" {
		return a.Name
	}
	return "@group:" + a.GroupRef
}

// xsdParticleName returns a stable identifier for a particle.
func xsdParticleName(p xsdParticle) string {
	switch t := p.(type) {
	case xsdElemParticle:
		if t.Elem.Ref != "" {
			return "@ref:" + t.Elem.Ref
		}
		return t.Elem.Name
	case xsdGroupRefParticle:
		return "@group:" + t.Ref
	case xsdAnyParticle:
		return "@any"
	case xsdCompParticle:
		return "@" + t.Comp.Kind
	}
	return "@unknown"
}

// xsdParticlesMatch returns true when two particles have the same
// identity.
func xsdParticlesMatch(a, b xsdParticle) bool {
	return xsdParticleName(a) == xsdParticleName(b)
}

// xsdMinOccurs returns the effective minOccurs for a particle.
func xsdMinOccurs(p xsdParticle) int {
	switch t := p.(type) {
	case xsdElemParticle:
		return t.Elem.Occurs.Min
	case xsdGroupRefParticle:
		return t.Occurs.Min
	case xsdAnyParticle:
		return t.Occurs.Min
	case xsdCompParticle:
		return t.Comp.Occurs.Min
	}
	return 0
}

// xsdOccursStr formats a maxOccurs value for error messages.
func xsdOccursStr(n int) string {
	if n == -1 {
		return "unbounded"
	}
	return strconv.Itoa(n)
}

// xsdParseIntFacet parses an integer facet value; returns 0 on
// failure.
func xsdParseIntFacet(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}

// xsdStrSliceEq reports whether two string slices are identical.
func xsdStrSliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// xsdConsumeToEnd discards tokens until the end element matching
// localname is consumed. Used to finish partially-decoded elements.
func xsdConsumeToEnd(dec *xml.Decoder, localname string) {
	depth := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			return
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			if depth == 0 &&
				t.Name.Local == localname {
				return
			}
			depth--
		}
	}
}
