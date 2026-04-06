// Package registry - Protobuf format compatibility checker.
//
// IsValid verifies that a version's document is syntactically valid
// Protobuf IDL (proto2 or proto3).
//
// IsCompatible checks whether two Protobuf schema versions are
// compatible in the given direction. Rules follow the closed-world
// assumption standard for schema registries:
//
//	"backward" — consumers using the NEW schema can read messages
//	             produced with the OLD schema.
//	             Permitted changes to the schema:
//	               • Add a new message, enum, or service.
//	               • Add a new optional field (absent in old messages
//	                 → reader uses the proto default).
//	               • Remove a field, provided its field number and
//	                 name are both marked reserved in the new schema
//	                 (prevents accidental number reuse).
//	             Forbidden changes:
//	               • Remove a message, enum, or service.
//	               • Remove a field without reserving its number/name.
//	               • Change a field's type to a wire-incompatible type.
//	             Implemented as: old ⊆ new (checkFileCompat(old, new))
//
//	"forward"  — consumers using the OLD schema can read messages
//	             produced with the NEW schema.
//	             Permitted changes to the schema:
//	               • Add a new message, enum, or service.
//	               • Add a new optional field (old consumers treat
//	                 unknown field numbers as unknown and preserve
//	                 them, so old schema consumers are unaffected).
//	             Forbidden changes:
//	               • Remove any field (new messages lack it; old
//	                 consumers that reference it get a zero default
//	                 which may be incorrect).
//	               • Remove a message, enum, or service.
//	               • Change a field's type to a wire-incompatible type.
//	             Implemented as: new ⊆ old (checkFileCompat(new, old))
//	             (forward compat = backward compat with args swapped)
//
// Compatibility checks – status per construct:
//
// Top-level
//   - [supported]     package (must not change)
//   - [supported]     message (removal detected; field-level compat
//     checked recursively)
//   - [supported]     enum (removal detected; value-level compat
//     checked)
//   - [supported]     service / rpc (removal detected; streaming
//     flags and message types checked)
//
// Message keywords
//   - [supported]     field number (immutable)
//   - [supported]     field type (wire-compatibility groups checked)
//   - [supported]     repeated ↔ singular (wire-safe for string /
//     bytes / message only)
//   - [supported]     reserved ranges and names (required when
//     removing a field)
//   - [supported]     oneof (move between oneofs detected)
//   - [supported]     nested messages and enums (recursive)
//   - [supported]     map fields (key and value type checked)
//
// Known limitations
//   - proto2 required fields are treated the same as optional.
//   - Extensions and options are not checked.
//   - Field renames are not detected (only field numbers are matched).

package registry

import (
	"bytes"
	"fmt"
	"io"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/desc/protoparse"
	. "github.com/xregistry/server/common"
	"google.golang.org/protobuf/types/descriptorpb"
)

func init() {
	RegisterFormat("protobuf", FormatProtobuf{})
}

type FormatProtobuf struct{}

// IsValid checks if the version is a valid Protobuf schema syntax.
func (fp FormatProtobuf) IsValid(version *Version) *XRError {
	buf := []byte(nil)

	if bufAny := version.Get(version.Resource.Singular); !IsNil(bufAny) {
		buf = bufAny.([]byte)
	}

	if len(buf) == 0 {
		return NewXRError("bad_request", version.XID,
			"error_detail="+version.XID+"is not a valid protobuf file")
	}

	fn := func(f string) (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(buf)), nil
	}
	p := protoparse.Parser{
		Accessor: protoparse.FileAccessor(fn),
	}
	_, err := p.ParseFiles("schema.proto")

	if err != nil {
		return NewXRError("bad_request", version.XID,
			"error_detail="+version.XID+
				"is not a valid protobuf file:"+err.Error())
	}
	return nil
}

// IsValidProto returns nil when buf is a syntactically valid Protobuf
// IDL file, or an error describing the syntax problem.
func IsValidProto(buf []byte) error {
	_, err := parseProto(buf)
	return err
}

// checks if both buffers are valid Protobuf schemas and whether
// newVersion is compatible with oldVersion in the given direction.
func (fp FormatProtobuf) IsCompatible(
	direction string,
	oldVersion *Version,
	newVersion *Version,
) *XRError {
	oldBuf, newBuf := []byte(nil), []byte(nil)

	if bufAny := oldVersion.Get(oldVersion.Resource.Singular); !IsNil(bufAny) {
		oldBuf = bufAny.([]byte)
	}
	if bufAny := newVersion.Get(newVersion.Resource.Singular); !IsNil(bufAny) {
		newBuf = bufAny.([]byte)
	}

	if len(oldBuf) == 0 {
		return NewXRError("bad_request", oldVersion.XID,
			"error_detail="+oldVersion.XID+"is not a valid protobuf file")
	}
	if len(newBuf) == 0 {
		return NewXRError("bad_request", newVersion.XID,
			"error_detail="+newVersion.XID+"is not a valid protobuf file")
	}

	oldDesc, err := parseProto(oldBuf)
	if err != nil {
		return NewXRError("bad_request", oldVersion.XID,
			"error_detail="+oldVersion.XID+
				"is not a valid protobuf file: "+err.Error())
	}
	newDesc, err := parseProto(newBuf)
	if err != nil {
		return NewXRError("bad_request", oldVersion.XID,
			"error_detail="+oldVersion.XID+
				"is not a valid protobuf file: "+err.Error())
	}

	// "forward" is backward with args swapped: new ⊆ old.
	checkOld, checkNew := oldDesc, newDesc
	if direction == "forward" {
		checkOld, checkNew = newDesc, oldDesc
	}

	err = checkFileCompat(checkOld, checkNew)
	if err != nil {
		compat := newVersion.
			Resource.
			MustFindMeta(false, FOR_READ).
			GetAsString("compatibility")

		return NewXRError("bad_request", newVersion.XID,
			"error_detail="+
				fmt.Sprintf("Version %q isn't %q compatible with %q: %s",
					newVersion.XID, compat, oldVersion.XID, err.Error()))
	}
	return nil
}

func parseProto(buf []byte) (*desc.FileDescriptor, error) {
	p := protoparse.Parser{
		Accessor: protoparse.FileAccessor(func(filename string) (io.ReadCloser, error) {
			if filename == "schema.proto" {
				return io.NopCloser(bytes.NewReader(buf)), nil
			}
			return nil, fmt.Errorf("file not found: %s", filename)
		}),
	}
	fds, err := p.ParseFiles("schema.proto")
	if err != nil {
		return nil, err
	}
	if len(fds) != 1 {
		return nil, fmt.Errorf("expected exactly one file descriptor, got %d",
			len(fds))
	}
	return fds[0], nil
}

func checkFileCompat(oldD, newD *desc.FileDescriptor) error {
	if oldD.GetPackage() != newD.GetPackage() {
		return fmt.Errorf("package changed from %q to %q",
			oldD.GetPackage(), newD.GetPackage())
	}

	// Enums (top-level)
	oldEnums := oldD.GetEnumTypes()
	newEnumsMap := make(map[string]*desc.EnumDescriptor)
	for _, e := range newD.GetEnumTypes() {
		newEnumsMap[e.GetName()] = e
	}
	for _, oldE := range oldEnums {
		newE, ok := newEnumsMap[oldE.GetName()]
		if !ok {
			return fmt.Errorf("enum %q removed", oldE.GetFullyQualifiedName())
		}
		if err := checkEnumCompat(oldE, newE); err != nil {
			return err
		}
	}

	// Messages (top-level)
	oldMsgs := oldD.GetMessageTypes()
	newMsgsMap := make(map[string]*desc.MessageDescriptor)
	for _, m := range newD.GetMessageTypes() {
		newMsgsMap[m.GetName()] = m
	}
	for _, oldM := range oldMsgs {
		newM, ok := newMsgsMap[oldM.GetName()]
		if !ok {
			return fmt.Errorf("message %q removed",
				oldM.GetFullyQualifiedName())
		}
		if err := checkMessageCompat(oldM, newM); err != nil {
			return err
		}
	}

	// Services
	oldSvcs := oldD.GetServices()
	newSvcsMap := make(map[string]*desc.ServiceDescriptor)
	for _, s := range newD.GetServices() {
		newSvcsMap[s.GetName()] = s
	}
	for _, oldS := range oldSvcs {
		newS, ok := newSvcsMap[oldS.GetName()]
		if !ok {
			return fmt.Errorf("service %q removed",
				oldS.GetFullyQualifiedName())
		}
		if err := checkServiceCompat(oldS, newS); err != nil {
			return err
		}
	}

	return nil
}

func checkEnumCompat(oldE, newE *desc.EnumDescriptor) error {
	oldVals := oldE.GetValues()
	newValsMap := make(map[int32]*desc.EnumValueDescriptor)
	for _, v := range newE.GetValues() {
		newValsMap[v.GetNumber()] = v
	}
	for _, oldV := range oldVals {
		newV, ok := newValsMap[oldV.GetNumber()]
		if !ok {
			return fmt.Errorf("enum value number %d (%q) removed from enum %q",
				oldV.GetNumber(), oldV.GetName(), oldE.GetFullyQualifiedName())
		}
		if oldV.GetName() != newV.GetName() {
			return fmt.Errorf("enum value name for number %d changed "+
				"from %q to %q in enum %q",
				oldV.GetNumber(), oldV.GetName(), newV.GetName(),
				oldE.GetFullyQualifiedName())
		}
	}
	return nil
}

func checkMessageCompat(oldM, newM *desc.MessageDescriptor) error {
	oldFieldsByNum := make(map[int32]*desc.FieldDescriptor)
	for _, f := range oldM.GetFields() {
		oldFieldsByNum[int32(f.GetNumber())] = f
	}

	newFieldsByNum := make(map[int32]*desc.FieldDescriptor)
	for _, f := range newM.GetFields() {
		newFieldsByNum[int32(f.GetNumber())] = f
	}

	// Reserved ranges and names – via underlying DescriptorProto (compatible
	// with all versions)
	protoMsg := newM.AsDescriptorProto()
	reservedRanges := protoMsg.GetReservedRange()

	newReservedNames := make(map[string]struct{})
	for _, name := range protoMsg.GetReservedName() {
		newReservedNames[name] = struct{}{}
	}

	for num, oldF := range oldFieldsByNum {
		newF, ok := newFieldsByNum[num]
		if !ok {
			// Field removed → must be reserved
			isReserved := false
			for _, r := range reservedRanges {
				if r.GetStart() <= num && num <= r.GetEnd() {
					isReserved = true
					break
				}
			}
			if !isReserved {
				return fmt.Errorf("field number %d (%q) removed from "+
					"message %q without being reserved",
					num, oldF.GetName(), oldM.GetFullyQualifiedName())
			}
			if _, reserved := newReservedNames[oldF.GetName()]; !reserved {
				return fmt.Errorf("field name %q removed from message %q "+
					"without being reserved",
					oldF.GetName(), oldM.GetFullyQualifiedName())
			}
			continue
		}

		if err := checkFieldCompat(oldF, newF); err != nil {
			return fmt.Errorf("in message %q, field %q: %v",
				oldM.GetFullyQualifiedName(), oldF.GetName(), err)
		}
	}

	// Nested messages
	oldNested := oldM.GetNestedMessageTypes()
	newNestedMap := make(map[string]*desc.MessageDescriptor)
	for _, nm := range newM.GetNestedMessageTypes() {
		newNestedMap[nm.GetName()] = nm
	}
	for _, oldNM := range oldNested {
		newNM, ok := newNestedMap[oldNM.GetName()]
		if !ok {
			return fmt.Errorf("nested message %q removed from %q",
				oldNM.GetName(), oldM.GetFullyQualifiedName())
		}
		if err := checkMessageCompat(oldNM, newNM); err != nil {
			return err
		}
	}

	// Nested enums
	oldNestedEnums := oldM.GetNestedEnumTypes()
	newNestedEnumsMap := make(map[string]*desc.EnumDescriptor)
	for _, ne := range newM.GetNestedEnumTypes() {
		newNestedEnumsMap[ne.GetName()] = ne
	}
	for _, oldNE := range oldNestedEnums {
		newNE, ok := newNestedEnumsMap[oldNE.GetName()]
		if !ok {
			return fmt.Errorf("nested enum %q removed from %q",
				oldNE.GetName(), oldM.GetFullyQualifiedName())
		}
		if err := checkEnumCompat(oldNE, newNE); err != nil {
			return err
		}
	}

	return nil
}

func checkFieldCompat(oldF, newF *desc.FieldDescriptor) error {
	if oldF.GetNumber() != newF.GetNumber() {
		return fmt.Errorf("field number changed from %d to %d",
			oldF.GetNumber(), newF.GetNumber())
	}

	if oldF.IsRepeated() != newF.IsRepeated() {
		if !isRepeatedSingularCompatible(oldF.GetType()) {
			return fmt.Errorf("repeated ↔ singular change not allowed for "+
				"type %v", oldF.GetType())
		}
	}

	if oldF.GetType() != newF.GetType() {
		if !areTypesCompatible(oldF.GetType(), newF.GetType()) {
			return fmt.Errorf("type changed from %v to %v (incompatible)",
				oldF.GetType(), newF.GetType())
		}
	}

	// Recurse into message/enum types
	if oldF.GetType() == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE {
		if err := checkMessageCompat(oldF.GetMessageType(),
			newF.GetMessageType()); err != nil {
			return err
		}
	}
	if oldF.GetType() == descriptorpb.FieldDescriptorProto_TYPE_ENUM {
		if err := checkEnumCompat(oldF.GetEnumType(),
			newF.GetEnumType()); err != nil {
			return err
		}
	}

	// Map fields
	if oldF.IsMap() != newF.IsMap() {
		return fmt.Errorf("map field status changed")
	}
	if oldF.IsMap() {
		if oldF.GetMapKeyType().GetType() != newF.GetMapKeyType().GetType() {
			return fmt.Errorf("map key type changed")
		}
		if err := checkFieldCompat(oldF.GetMapValueType(),
			newF.GetMapValueType()); err != nil {
			return err
		}
	}

	// Oneof handling (basic)
	oldOneof := oldF.GetOneOf()
	newOneof := newF.GetOneOf()
	if oldOneof != nil && newOneof == nil {
		// Moving from oneof → standalone: safe only if old oneof had 1 field
		if len(oldOneof.GetChoices()) > 1 {
			return fmt.Errorf("field moved from multi-field oneof to " +
				"standalone")
		}
	} else if oldOneof == nil && newOneof != nil {
		// Standalone → oneof: usually safe
	} else if oldOneof != nil && newOneof != nil {
		if oldOneof.GetName() != newOneof.GetName() {
			return fmt.Errorf("field moved to a different oneof")
		}
	}

	return nil
}

func isRepeatedSingularCompatible(t descriptorpb.FieldDescriptorProto_Type) bool {
	switch t {
	case descriptorpb.FieldDescriptorProto_TYPE_STRING,
		descriptorpb.FieldDescriptorProto_TYPE_BYTES,
		descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
		return true
	default:
		return false
	}
}

func areTypesCompatible(oldT, newT descriptorpb.FieldDescriptorProto_Type) bool {
	intTypes := []descriptorpb.FieldDescriptorProto_Type{
		descriptorpb.FieldDescriptorProto_TYPE_INT32,
		descriptorpb.FieldDescriptorProto_TYPE_UINT32,
		descriptorpb.FieldDescriptorProto_TYPE_INT64,
		descriptorpb.FieldDescriptorProto_TYPE_UINT64,
		descriptorpb.FieldDescriptorProto_TYPE_BOOL,
	}
	sintTypes := []descriptorpb.FieldDescriptorProto_Type{
		descriptorpb.FieldDescriptorProto_TYPE_SINT32,
		descriptorpb.FieldDescriptorProto_TYPE_SINT64,
	}
	fixed32Types := []descriptorpb.FieldDescriptorProto_Type{
		descriptorpb.FieldDescriptorProto_TYPE_FIXED32,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED32,
	}
	fixed64Types := []descriptorpb.FieldDescriptorProto_Type{
		descriptorpb.FieldDescriptorProto_TYPE_FIXED64,
		descriptorpb.FieldDescriptorProto_TYPE_SFIXED64,
	}
	stringBytes := []descriptorpb.FieldDescriptorProto_Type{
		descriptorpb.FieldDescriptorProto_TYPE_STRING,
		descriptorpb.FieldDescriptorProto_TYPE_BYTES,
	}

	switch {
	case contains(intTypes, oldT):
		return contains(intTypes, newT)
	case contains(sintTypes, oldT):
		return contains(sintTypes, newT)
	case contains(fixed32Types, oldT):
		return contains(fixed32Types, newT)
	case contains(fixed64Types, oldT):
		return contains(fixed64Types, newT)
	case contains(stringBytes, oldT):
		return contains(stringBytes, newT)
	case oldT == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE:
		return newT == descriptorpb.FieldDescriptorProto_TYPE_MESSAGE ||
			newT == descriptorpb.FieldDescriptorProto_TYPE_BYTES
	default:
		return oldT == newT
	}
}

func contains(slice []descriptorpb.FieldDescriptorProto_Type, item descriptorpb.FieldDescriptorProto_Type) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func checkServiceCompat(oldS, newS *desc.ServiceDescriptor) error {
	oldMethods := oldS.GetMethods()
	newMethodsMap := make(map[string]*desc.MethodDescriptor)
	for _, m := range newS.GetMethods() {
		newMethodsMap[m.GetName()] = m
	}

	for _, oldM := range oldMethods {
		newM, ok := newMethodsMap[oldM.GetName()]
		if !ok {
			return fmt.Errorf("method %q removed from service %q",
				oldM.GetName(), oldS.GetFullyQualifiedName())
		}

		if oldM.IsClientStreaming() != newM.IsClientStreaming() ||
			oldM.IsServerStreaming() != newM.IsServerStreaming() {
			return fmt.Errorf("streaming configuration changed for method "+
				"%q in service %q",
				oldM.GetName(), oldS.GetFullyQualifiedName())
		}

		err := checkMessageCompat(oldM.GetInputType(), newM.GetInputType())
		if err != nil {
			return fmt.Errorf("input type not compatible for method %q: %v",
				oldM.GetName(), err)
		}
		err = checkMessageCompat(oldM.GetOutputType(), newM.GetOutputType())
		if err != nil {
			return fmt.Errorf("output type not compatible for method %q: %v",
				oldM.GetName(), err)
		}
	}

	return nil
}
