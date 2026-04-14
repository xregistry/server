package registry

// Unit tests for Protobuf IsValid (via IsValidProto) and
// IsCompatible (via checkFileCompat / direction dispatch).
//
// Tests are run with: make utest

import (
	"testing"
)

// ── Helpers ────────────────────────────────────────────────────────

// protoCompat wraps checkFileCompat with direction handling,
// mirroring what FormatProtobuf.IsCompatible does.
func protoCompat(
	direction, oldProto, newProto string,
) error {
	oldD, err := parseProto([]byte(oldProto))
	if err != nil {
		return err
	}
	newD, err := parseProto([]byte(newProto))
	if err != nil {
		return err
	}
	checkOld, checkNew := oldD, newD
	if direction == "forward" {
		checkOld, checkNew = newD, oldD
	}
	return checkFileCompat(checkOld, checkNew)
}

type protoCase struct {
	name    string
	dir     string
	old     string
	new     string
	wantErr bool
}

func runProtoCases(t *testing.T, cases []protoCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := protoCompat(tc.dir, tc.old, tc.new)
			if tc.wantErr && err == nil {
				t.Errorf("expected incompatibility, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected incompatibility: %v", err)
			}
		})
	}
}

// ── IsValidProto ───────────────────────────────────────────────────

func TestIsValidProto(t *testing.T) {
	tests := []struct {
		name    string
		schema  string
		wantErr bool
	}{
		// ── Valid ──────────────────────────────────────────────────
		{
			name: "minimal proto3",
			schema: `syntax = "proto3";
message Empty {}`,
		},
		{
			name: "proto3 with scalar fields",
			schema: `syntax = "proto3";
message Person {
  string name = 1;
  int32  age  = 2;
  bool   active = 3;
}`,
		},
		{
			name: "proto3 with enum",
			schema: `syntax = "proto3";
enum Status { UNKNOWN = 0; ACTIVE = 1; INACTIVE = 2; }
message Item { Status status = 1; }`,
		},
		{
			name: "proto3 with nested message",
			schema: `syntax = "proto3";
message Outer {
  message Inner { string value = 1; }
  Inner inner = 1;
}`,
		},
		{
			name: "proto3 with map field",
			schema: `syntax = "proto3";
message Config { map<string, string> labels = 1; }`,
		},
		{
			name: "proto3 with oneof",
			schema: `syntax = "proto3";
message Msg {
  oneof body { string text = 1; bytes data = 2; }
}`,
		},
		{
			name: "proto3 with repeated field",
			schema: `syntax = "proto3";
message List { repeated string items = 1; }`,
		},
		{
			name: "proto3 with service",
			schema: `syntax = "proto3";
message Req {}
message Res {}
service Greeter { rpc Hello(Req) returns (Res); }`,
		},
		{
			name: "proto3 with reserved fields",
			schema: `syntax = "proto3";
message Msg {
  string name = 1;
  reserved 2, 3;
  reserved "old_field";
}`,
		},
		{
			name: "proto3 with package",
			schema: `syntax = "proto3";
package com.example;
message Ping {}`,
		},

		// ── Invalid ────────────────────────────────────────────────
		{
			name:    "empty input",
			schema:  `not proto at all { broken`,
			wantErr: true,
		},
		{
			name:    "not protobuf",
			schema:  `{"type":"record"}`,
			wantErr: true,
		},
		{
			name: "missing field number",
			schema: `syntax = "proto3";
message Bad { string name; }`,
			wantErr: true,
		},
		{
			name: "duplicate field numbers",
			schema: `syntax = "proto3";
message Bad { string a = 1; int32 b = 1; }`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := IsValidProto([]byte(tc.schema))
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// ── Package ────────────────────────────────────────────────────────

func TestProtoCompat_Package(t *testing.T) {
	runProtoCases(t, []protoCase{
		{
			name: "same package – compat",
			dir:  "backward",
			old: `syntax="proto3"; package com.example;
				message M { string x = 1; }`,
			new: `syntax="proto3"; package com.example;
				message M { string x = 1; }`,
		},
		{
			name:    "package changed – incompatible",
			dir:     "backward",
			old:     `syntax="proto3"; package a; message M {}`,
			new:     `syntax="proto3"; package b; message M {}`,
			wantErr: true,
		},
	})
}

// ── Adding / removing messages ─────────────────────────────────────

func TestProtoCompat_Messages(t *testing.T) {
	runProtoCases(t, []protoCase{
		{
			name: "add new message – compat (backward)",
			dir:  "backward",
			old:  `syntax="proto3"; message A { string x = 1; }`,
			new: `syntax="proto3";
				message A { string x = 1; }
				message B { int32 y = 1; }`,
		},
		{
			name:    "remove message – incompatible (backward)",
			dir:     "backward",
			old:     `syntax="proto3"; message A {} message B {}`,
			new:     `syntax="proto3"; message A {}`,
			wantErr: true,
		},
		{
			// Forward = checkFileCompat(new, old): each new message
			// must exist in old. New adds B which old lacks → error.
			name: "add new message – incompatible (forward, conservative)",
			dir:  "forward",
			old:  `syntax="proto3"; message A { string x = 1; }`,
			new: `syntax="proto3";
				message A { string x = 1; }
				message B { int32 y = 1; }`,
			wantErr: true,
		},
		{
			// Forward = checkFileCompat(new, old): iterate over new
			// messages only. B is in old but not in new – not
			// iterated – so no error. Old consumers reading new data
			// see absent B fields as zero defaults.
			name: "remove message – compat (forward)",
			dir:  "forward",
			old:  `syntax="proto3"; message A {} message B {}`,
			new:  `syntax="proto3"; message A {}`,
		},
	})
}

// ── Adding / removing fields ───────────────────────────────────────

func TestProtoCompat_Fields(t *testing.T) {
	runProtoCases(t, []protoCase{
		{
			name: "add optional field – compat (backward)",
			dir:  "backward",
			old:  `syntax="proto3"; message M { string name = 1; }`,
			new: `syntax="proto3";
				message M { string name = 1; int32 age = 2; }`,
		},
		{
			name: "remove field with reservation – compat (backward)",
			dir:  "backward",
			old: `syntax="proto3";
				message M { string name = 1; int32 age = 2; }`,
			new: `syntax="proto3";
				message M {
				  string name = 1;
				  reserved 2; reserved "age";
				}`,
		},
		{
			name: "remove field without reservation – incompatible",
			dir:  "backward",
			old: `syntax="proto3";
				message M { string name = 1; int32 age = 2; }`,
			new:     `syntax="proto3"; message M { string name = 1; }`,
			wantErr: true,
		},
		{
			// Forward: new adds a field – old consumers see it as
			// unknown, which is fine; but checkFileCompat(new, old)
			// asks "does each new field exist in old?" – it doesn't,
			// so this is rejected as conservative forward compat.
			name:    "add field – incompatible (forward, conservative)",
			dir:     "forward",
			old:     `syntax="proto3"; message M { string name = 1; }`,
			new:     `syntax="proto3"; message M { string name = 1; int32 age = 2; }`,
			wantErr: true,
		},
		{
			name: "remove field with reservation – compat (forward)",
			dir:  "forward",
			old: `syntax="proto3";
				message M { string name = 1; int32 age = 2; }`,
			new: `syntax="proto3";
				message M {
				  string name = 1;
				  reserved 2; reserved "age";
				}`,
		},
	})
}

// ── Field type changes ─────────────────────────────────────────────

func TestProtoCompat_FieldTypes(t *testing.T) {
	runProtoCases(t, []protoCase{
		{
			name: "int32 → uint32 (same wire group) – compat",
			dir:  "backward",
			old:  `syntax="proto3"; message M { int32 x = 1; }`,
			new:  `syntax="proto3"; message M { uint32 x = 1; }`,
		},
		{
			name: "int32 → int64 (same wire group) – compat",
			dir:  "backward",
			old:  `syntax="proto3"; message M { int32 x = 1; }`,
			new:  `syntax="proto3"; message M { int64 x = 1; }`,
		},
		{
			name: "string → bytes (compatible) – compat",
			dir:  "backward",
			old:  `syntax="proto3"; message M { string x = 1; }`,
			new:  `syntax="proto3"; message M { bytes x = 1; }`,
		},
		{
			name:    "int32 → string – incompatible",
			dir:     "backward",
			old:     `syntax="proto3"; message M { int32 x = 1; }`,
			new:     `syntax="proto3"; message M { string x = 1; }`,
			wantErr: true,
		},
		{
			name:    "int32 → double – incompatible",
			dir:     "backward",
			old:     `syntax="proto3"; message M { int32 x = 1; }`,
			new:     `syntax="proto3"; message M { double x = 1; }`,
			wantErr: true,
		},
		{
			name: "fixed32 ↔ sfixed32 (same group) – compat",
			dir:  "backward",
			old:  `syntax="proto3"; message M { fixed32 x = 1; }`,
			new:  `syntax="proto3"; message M { sfixed32 x = 1; }`,
		},
		{
			name: "fixed64 ↔ sfixed64 (same group) – compat",
			dir:  "backward",
			old:  `syntax="proto3"; message M { fixed64 x = 1; }`,
			new:  `syntax="proto3"; message M { sfixed64 x = 1; }`,
		},
		{
			name: "sint32 ↔ sint64 (same group) – compat",
			dir:  "backward",
			old:  `syntax="proto3"; message M { sint32 x = 1; }`,
			new:  `syntax="proto3"; message M { sint64 x = 1; }`,
		},
	})
}

// ── Repeated / singular ────────────────────────────────────────────

func TestProtoCompat_Repeated(t *testing.T) {
	runProtoCases(t, []protoCase{
		{
			name: "singular string → repeated string – compat",
			dir:  "backward",
			old:  `syntax="proto3"; message M { string x = 1; }`,
			new:  `syntax="proto3"; message M { repeated string x = 1; }`,
		},
		{
			name: "repeated string → singular string – compat",
			dir:  "backward",
			old:  `syntax="proto3"; message M { repeated string x = 1; }`,
			new:  `syntax="proto3"; message M { string x = 1; }`,
		},
		{
			name:    "singular int32 → repeated int32 – incompatible",
			dir:     "backward",
			old:     `syntax="proto3"; message M { int32 x = 1; }`,
			new:     `syntax="proto3"; message M { repeated int32 x = 1; }`,
			wantErr: true,
		},
	})
}

// ── Enums ──────────────────────────────────────────────────────────

func TestProtoCompat_Enums(t *testing.T) {
	runProtoCases(t, []protoCase{
		{
			name: "add enum value – compat (backward)",
			dir:  "backward",
			old: `syntax="proto3";
				enum E { UNKNOWN = 0; ACTIVE = 1; }`,
			new: `syntax="proto3";
				enum E { UNKNOWN = 0; ACTIVE = 1; RETIRED = 2; }`,
		},
		{
			name: "same enum – compat",
			dir:  "backward",
			old:  `syntax="proto3"; enum E { UNKNOWN=0; A=1; }`,
			new:  `syntax="proto3"; enum E { UNKNOWN=0; A=1; }`,
		},
		{
			name: "remove enum value – incompatible (backward)",
			dir:  "backward",
			old: `syntax="proto3";
				enum E { UNKNOWN = 0; ACTIVE = 1; RETIRED = 2; }`,
			new: `syntax="proto3";
				enum E { UNKNOWN = 0; ACTIVE = 1; }`,
			wantErr: true,
		},
		{
			name:    "rename enum value – incompatible",
			dir:     "backward",
			old:     `syntax="proto3"; enum E { UNKNOWN=0; OLD=1; }`,
			new:     `syntax="proto3"; enum E { UNKNOWN=0; NEW=1; }`,
			wantErr: true,
		},
		{
			name:    "remove enum – incompatible",
			dir:     "backward",
			old:     `syntax="proto3"; enum E { A=0; } message M {}`,
			new:     `syntax="proto3"; message M {}`,
			wantErr: true,
		},
	})
}

// ── Nested messages ────────────────────────────────────────────────

func TestProtoCompat_Nested(t *testing.T) {
	runProtoCases(t, []protoCase{
		{
			name: "add field to nested message – compat (backward)",
			dir:  "backward",
			old: `syntax="proto3";
				message Outer {
				  message Inner { string v = 1; }
				  Inner inner = 1;
				}`,
			new: `syntax="proto3";
				message Outer {
				  message Inner { string v = 1; int32 n = 2; }
				  Inner inner = 1;
				}`,
		},
		{
			name: "remove nested message – incompatible",
			dir:  "backward",
			old: `syntax="proto3";
				message Outer {
				  message Inner {}
				  message Other {}
				}`,
			new: `syntax="proto3";
				message Outer {
				  message Inner {}
				}`,
			wantErr: true,
		},
	})
}

// ── Services ───────────────────────────────────────────────────────

func TestProtoCompat_Services(t *testing.T) {
	runProtoCases(t, []protoCase{
		{
			name: "add service method – compat (backward)",
			dir:  "backward",
			old: `syntax="proto3";
				message Req {} message Res {}
				service S { rpc A(Req) returns (Res); }`,
			new: `syntax="proto3";
				message Req {} message Res {}
				service S {
				  rpc A(Req) returns (Res);
				  rpc B(Req) returns (Res);
				}`,
		},
		{
			name: "remove service method – incompatible",
			dir:  "backward",
			old: `syntax="proto3";
				message Req {} message Res {}
				service S {
				  rpc A(Req) returns (Res);
				  rpc B(Req) returns (Res);
				}`,
			new: `syntax="proto3";
				message Req {} message Res {}
				service S { rpc A(Req) returns (Res); }`,
			wantErr: true,
		},
		{
			name: "remove service – incompatible",
			dir:  "backward",
			old: `syntax="proto3";
				message Req {} message Res {}
				service S { rpc A(Req) returns (Res); }`,
			new:     `syntax="proto3"; message Req {} message Res {}`,
			wantErr: true,
		},
		{
			name: "change streaming flag – incompatible",
			dir:  "backward",
			old: `syntax="proto3";
				message Req {} message Res {}
				service S { rpc A(Req) returns (Res); }`,
			new: `syntax="proto3";
				message Req {} message Res {}
				service S {
				  rpc A(Req) returns (stream Res);
				}`,
			wantErr: true,
		},
	})
}

// ── Forward direction ──────────────────────────────────────────────

func TestProtoCompat_ForwardDirection(t *testing.T) {
	runProtoCases(t, []protoCase{
		{
			name: "remove field with reservation – compat (forward)",
			dir:  "forward",
			old: `syntax="proto3";
				message M { string name = 1; int32 age = 2; }`,
			new: `syntax="proto3";
				message M {
				  string name = 1;
				  reserved 2; reserved "age";
				}`,
		},
		{
			// Forward = checkFileCompat(new, old): only new fields
			// are iterated. Age was removed from new so it is never
			// checked against old → no error. Old consumers reading
			// new data see absent "age" as proto default (0).
			name: "remove field without reservation – compat (forward)",
			dir:  "forward",
			old: `syntax="proto3";
				message M { string name = 1; int32 age = 2; }`,
			new: `syntax="proto3"; message M { string name = 1; }`,
		},
		{
			name: "add field – incompatible (conservative)",
			dir:  "forward",
			old:  `syntax="proto3"; message M { string name = 1; }`,
			new: `syntax="proto3";
				message M { string name = 1; int32 age = 2; }`,
			wantErr: true,
		},
		{
			name:    "incompatible type change – incompatible (forward)",
			dir:     "forward",
			old:     `syntax="proto3"; message M { int32 x = 1; }`,
			new:     `syntax="proto3"; message M { string x = 1; }`,
			wantErr: true,
		},
	})
}
