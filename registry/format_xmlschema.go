// Package registry - XML Schema format compatibility checker.
//
// IsValid verifies that a version's document is a syntactically valid
// XML Schema (XSD).
//
// IsCompatible checks whether two XML Schema versions are compatible in
// the given direction using a conservative subset rule over global
// declarations:
//
//   - "backward": old ⊆ new
//   - "forward":  new ⊆ old
//
// A declaration is considered compatible if a declaration with the same
// name exists and its normalized signature is unchanged.

package registry

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"

	. "github.com/xregistry/server/common"
)

const XMLSCHEMA_FORMAT = "xmlschema.*"

const xmlSchemaNamespace = "http://www.w3.org/2001/XMLSchema"

func init() {
	RegisterFormat(XMLSCHEMA_FORMAT, FormatXMLSchema{})
}

// FormatXMLSchema implements the Format interface for XML Schema.
type FormatXMLSchema struct{}

func (fx FormatXMLSchema) IsValid(ver *Version) (string, *XRError) {
	format := ver.GetAsString("format")
	if ok, _ := regexp.MatchString("(?i)"+XMLSCHEMA_FORMAT, format); !ok {
		return "true", NewXRError("bad_request", ver.XID,
			"error_detail="+
				fmt.Sprintf(`Version %q has a "format" value of %q, was `+
					`expecting %q`, ver.XID, format, XMLSCHEMA_FORMAT))
	}

	if ver.Resource.ResourceModel.GetHasDocument() == false {
		return "true", NewXRError("format_violation", ver.XID,
			"format="+format).
			SetDetailf(`The Resource (%s) for Version %q does not have `+
				`"hasdocument" in its resource model set to "true", and an `+
				`empty/missing document is not compliant.`,
				ver.Resource.XID, ver.XID)
	}

	if resURL := ver.Get(ver.Resource.Singular + "url"); !IsNil(resURL) {
		return "false, data stored externally",
			NewXRError("format_external", ver.XID)
	}

	buf := []byte(nil)
	if bufAny := ver.Get(ver.Resource.Singular); !IsNil(bufAny) {
		buf = bufAny.([]byte)
	}

	if len(buf) == 0 {
		return "true", NewXRError("format_violation", ver.XID,
			"format="+ver.GetAsString("format")).
			SetDetailf("Version %q is empty and therefore not a "+
				"valid xml schema file.", ver.XID)
	}

	if err := IsValidXMLSchema(buf); err != nil {
		return "true", NewXRError("format_violation", ver.XID,
			"format="+ver.GetAsString("format")).
			SetDetailf("Version %q is not a valid xml schema file: %s.",
				ver.XID, err)
	}

	return "true", nil
}

func (fx FormatXMLSchema) IsCompatible(
	direction string,
	oldVersion *Version,
	newVersion *Version,
) (string, *XRError) {
	oldBuf, newBuf := []byte(nil), []byte(nil)

	reason, xErr := fx.IsValid(oldVersion)
	if xErr != nil {
		return reason, xErr
	}

	reason, xErr = fx.IsValid(newVersion)
	if xErr != nil {
		return reason, xErr
	}

	if b := oldVersion.Get(oldVersion.Resource.Singular); !IsNil(b) {
		oldBuf = b.([]byte)
	}

	if b := newVersion.Get(newVersion.Resource.Singular); !IsNil(b) {
		newBuf = b.([]byte)
	}

	if err := checkXMLSchemaCompat(direction, oldBuf, newBuf); err != nil {
		compat := newVersion.
			Resource.
			MustFindMeta(false, FOR_READ).
			GetAsString("compatibility")

		return "true", NewXRError("compatibility_violation", newVersion.XID,
			"compat="+compat).
			SetDetailf("Version %q isn't %q compatible with %q: %s",
				newVersion.XID, compat, oldVersion.XID,
				err.Error())
	}

	return "true", nil
}

// IsValidXMLSchema returns nil when buf is syntactically valid XML
// Schema, or an error describing the problem.
func IsValidXMLSchema(buf []byte) error {
	doc, err := parseXMLSchema(buf)
	if err != nil {
		return err
	}

	if doc.XMLName.Local != "schema" {
		return fmt.Errorf("root element must be schema")
	}

	ns := strings.TrimSpace(doc.XMLName.Space)
	if ns != "" && ns != xmlSchemaNamespace {
		return fmt.Errorf("unsupported schema namespace %q", ns)
	}

	return nil
}

type xmlSchemaDoc struct {
	XMLName      xml.Name               `xml:"schema"`
	Elements     []xmlSchemaDeclaration `xml:"element"`
	ComplexTypes []xmlSchemaDeclaration `xml:"complexType"`
	SimpleTypes  []xmlSchemaDeclaration `xml:"simpleType"`
}

type xmlSchemaDeclaration struct {
	Name  string `xml:"name,attr"`
	Type  string `xml:"type,attr"`
	Inner string `xml:",innerxml"`
}

func parseXMLSchema(buf []byte) (*xmlSchemaDoc, error) {
	var doc xmlSchemaDoc
	if err := xml.Unmarshal(buf, &doc); err != nil {
		return nil, err
	}
	if doc.XMLName.Local == "" {
		return nil, fmt.Errorf("missing schema root element")
	}
	return &doc, nil
}

func checkXMLSchemaCompat(direction string, oldBuf, newBuf []byte) error {
	oldDecls, err := schemaDecls(oldBuf)
	if err != nil {
		return fmt.Errorf("old schema invalid: %w", err)
	}

	newDecls, err := schemaDecls(newBuf)
	if err != nil {
		return fmt.Errorf("new schema invalid: %w", err)
	}

	checkOld, checkNew := oldDecls, newDecls
	if strings.EqualFold(direction, "forward") {
		checkOld, checkNew = newDecls, oldDecls
	}

	for key, oldSig := range checkOld {
		newSig, ok := checkNew[key]
		if !ok {
			return fmt.Errorf("declaration %q removed", key)
		}
		if oldSig != newSig {
			return fmt.Errorf("declaration %q changed", key)
		}
	}

	return nil
}

func schemaDecls(buf []byte) (map[string]string, error) {
	if err := IsValidXMLSchema(buf); err != nil {
		return nil, err
	}

	doc, err := parseXMLSchema(buf)
	if err != nil {
		return nil, err
	}

	decls := map[string]string{}
	add := func(kind string, d xmlSchemaDeclaration) error {
		name := strings.TrimSpace(d.Name)
		if name == "" {
			return nil
		}

		sig := strings.TrimSpace(d.Type) + "|" +
			compactXML(d.Inner)
		key := kind + ":" + name
		if prev, found := decls[key]; found && prev != sig {
			return fmt.Errorf("duplicate %s declaration for %q", kind, name)
		}
		decls[key] = sig
		return nil
	}

	for _, d := range doc.Elements {
		if err := add("element", d); err != nil {
			return nil, err
		}
	}
	for _, d := range doc.ComplexTypes {
		if err := add("complexType", d); err != nil {
			return nil, err
		}
	}
	for _, d := range doc.SimpleTypes {
		if err := add("simpleType", d); err != nil {
			return nil, err
		}
	}

	return decls, nil
}

func compactXML(s string) string {
	fields := strings.Fields(strings.TrimSpace(s))
	return strings.Join(fields, " ")
}
