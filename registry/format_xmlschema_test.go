package registry

import "testing"

func TestIsValidXMLSchema(t *testing.T) {
	tests := []struct {
		name    string
		schema  string
		wantErr bool
	}{
		{
			name: "minimal schema",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
</xs:schema>`,
		},
		{
			name: "schema with element and complex type",
			schema: `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="user" type="UserType"/>
  <xs:complexType name="UserType">
    <xs:sequence>
      <xs:element name="id" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`,
		},
		{
			name: "not xml",
			schema: `{
  "type": "object"
}`,
			wantErr: true,
		},
		{
			name: "wrong root",
			schema: `<root xmlns:xs="http://www.w3.org/2001/XMLSchema">
</root>`,
			wantErr: true,
		},
		{
			name: "wrong namespace",
			schema: `<xs:schema xmlns:xs="http://example.com/ns">
</xs:schema>`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := IsValidXMLSchema([]byte(tc.schema))
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCheckXMLSchemaCompat(t *testing.T) {
	base := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="id" type="xs:string"/>
  <xs:complexType name="UserType">
    <xs:sequence>
      <xs:element name="name" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	additive := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="id" type="xs:string"/>
  <xs:element name="email" type="xs:string"/>
  <xs:complexType name="UserType">
    <xs:sequence>
      <xs:element name="name" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	changed := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:element name="id" type="xs:int"/>
  <xs:complexType name="UserType">
    <xs:sequence>
      <xs:element name="name" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	removed := `<xs:schema xmlns:xs="http://www.w3.org/2001/XMLSchema">
  <xs:complexType name="UserType">
    <xs:sequence>
      <xs:element name="name" type="xs:string"/>
    </xs:sequence>
  </xs:complexType>
</xs:schema>`

	tests := []struct {
		name      string
		direction string
		old       string
		new       string
		wantErr   bool
	}{
		{
			name:      "same schema backward",
			direction: "backward",
			old:       base,
			new:       base,
		},
		{
			name:      "add declaration backward",
			direction: "backward",
			old:       base,
			new:       additive,
		},
		{
			name:      "remove declaration backward",
			direction: "backward",
			old:       base,
			new:       removed,
			wantErr:   true,
		},
		{
			name:      "change declaration backward",
			direction: "backward",
			old:       base,
			new:       changed,
			wantErr:   true,
		},
		{
			name:      "add declaration forward",
			direction: "forward",
			old:       base,
			new:       additive,
			wantErr:   true,
		},
		{
			name:      "remove declaration forward",
			direction: "forward",
			old:       base,
			new:       removed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkXMLSchemaCompat(
				tc.direction,
				[]byte(tc.old),
				[]byte(tc.new),
			)
			if tc.wantErr && err == nil {
				t.Errorf("expected incompatibility, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected incompatibility: %v", err)
			}
		})
	}
}
