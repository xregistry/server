package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"path"
	"regexp"
	"runtime"
	"strings"

	log "github.com/duglin/dlog"
)

type XRError struct {
	Type     string
	Code     int // HTTP response code
	Title    string
	Args     map[string]string
	Subject  string
	Detail   string
	Instance string
	Source   string
	Headers  map[string]string // HTTP headers to include in response
}

/*
func (xErr *XRError) Error() string {
	return xErr.ToJSON("")
}
*/

func NewXRError(daType string, subject string, args ...string) *XRError {
	err := Type2Error[daType]
	PanicIf(err == nil, "Unknown error type: %s", daType)

	PanicIf(subject != "" && !strings.HasPrefix(subject, "http") &&
		subject[0] != '/', "start with / : %s", subject)

	if daType == "server_error" {
		// This is special.
		// Only show internal info (args) to the logs.
		// Remove internal stuff from the error that's bubbled up.
		if len(args) > 0 {
			log.Printf("%v", args)
			args = nil
		}
		log.Printf(">>> System Error <<<")
		ShowStack()
	}
	// ShowStack()

	// Format "source" as "gitCommit:package:fileName:lineNumber"
	source := ""
	pc, file, lineNum, ok := runtime.Caller(1)
	if ok {
		pkg, _, _ := strings.Cut(path.Base(runtime.FuncForPC(pc).Name()), ".")
		file, _, _ = strings.Cut(path.Base(file), ".")
		source = fmt.Sprintf("%.12s:%s:%s:%d", GitCommit, pkg, file, lineNum)
	} else {
		source = fmt.Sprintf("%.12s", GitCommit)
	}

	tmpXErr := &XRError{
		Type:    err.Type,
		Subject: subject,
		Title:   err.Title,
		// Args:    argMap,
		Detail:  err.Detail,
		Code:    err.Code,
		Source:  source,
		Headers: maps.Clone(err.Headers),
	}

	return tmpXErr.SetArgs(args...)
}

var Type2Error = map[string]*XRError{
	// CODE SPEC
	"action_not_supported": &XRError{
		Code:  405,
		Title: `The specified action (<action>) is not supported for: <subject>.`,
	},
	"ancestor_circular_reference": &XRError{
		Code:  400,
		Title: `For "<subject>", the request would create a circular list of ancestors: <list>.`,
	},
	"bad_defaultversionid": &XRError{
		Code:  400,
		Title: `An error was found in the "defaultversionid" value specified (<value>): <error_detail>.`,
	},
	"bad_details": &XRError{
		Code:  400,
		Title: `Use of "$details" in this context is not allowed: <subject>.`,
	},
	"bad_filter": &XRError{
		Code:  400,
		Title: `An error was found in filter (<filter_name>): <error_detail>.`,
	},
	"bad_flag": &XRError{
		Code:  400,
		Title: `The specified flag (<flag>) is not allowed in this context: <subject>.`,
	},
	"bad_inline": &XRError{
		Code:  400,
		Title: `An error was found in inline value (<inline_value>): <error_detail>.`,
	},
	"bad_request": &XRError{
		Code:  400,
		Title: `<error_detail>.`,
	},
	"bad_sort": &XRError{
		Code:  400,
		Title: `An error was found in sort value (<sort_value>): <error_detail>.`,
	},
	"cannot_doc_xref": &XRError{
		Code:  400,
		Title: `Retrieving the document view of a Version for "<subject>" is not allowed because it uses "xref".`,
	},
	"capability_error": &XRError{
		Code:  400,
		Title: `There was an error in the capabilities provided: <error_detail>.`,
	},
	"capability_missing_specversion": &XRError{
		Code:  400,
		Title: `The "specversions" capability needs to contain "<value>".`,
	},
	"capability_unknown": &XRError{
		Code:  400,
		Title: `Unknown capability specified: <field>.`,
	},
	"capability_value": &XRError{
		Code:  400,
		Title: `Invalid value (<value>) specified for capability "<field>". Allowable values include: <list>.`,
	},
	"capability_wildcard": &XRError{
		Code:  400,
		Title: `When "<field>" includes a value of "*" then no other values are allowed.`,
	},
	"compatibility_violation": &XRError{
		Code:  400,
		Title: `The request would cause one or more Versions of "<subject>" to violate its compatibility rule (<compatibility_value>).`,
	},
	"data_retrieval_error": &XRError{
		Code:  500,
		Title: `The server was unable to retrieve all of the requested data.`,
	},
	"defaultversionid_request": &XRError{
		Code:  400,
		Title: `Processing "<subject>", the "defaultversionid" attribute is not allowed to be "request" since a Version wasn't processed.`,
	},
	"groups_only": &XRError{
		Code:  400,
		Title: `Attribute "<name>" is invalid. Only Group types are allowed to be specified on this request: <subject>.`,
	},
	"inline_noninlineable": &XRError{
		Code:  400,
		Title: `Attempting to inline a non-inlineable attribute (<name>) on: <subject>.`,
	},
	"invalid_attribute": &XRError{
		Code:  400,
		Title: `The attribute "<name>" for "<subject>" is not valid: <error_detail>.`,
	},
	"malformed_id": &XRError{
		Code:  400,
		Title: `The specified ID value (<id>) is malformed: <error_detail>.`,
	},
	"malformed_xid": &XRError{
		Code:  400,
		Title: `The specified XID value (<xid>) is malformed: <error_detail>.`,
	},
	"malformed_xref": &XRError{
		Code:  400,
		Title: `The specified xref value (<xref>) is malformed: <error_detail>.`,
	},
	"mismatched_epoch": &XRError{
		Code:  400,
		Title: `The specified epoch value (<bad_epoch>) for "<subject>" does not match its current value (<epoch>).`,
	},
	"mismatched_id": &XRError{
		Code:  400,
		Title: `The specified "<singular>id" value (<invalid_id>) for "<subject>" needs to be "<expected_id>".`,
	},
	"misplaced_epoch": &XRError{
		Code:  400,
		Title: `The specified "epoch" value for "<subject>" needs to be within a "meta" entity.`,
	},
	"missing_versions": &XRError{
		Code:  400,
		Title: `At least one Version needs to be included in the request to process "<subject>".`,
	},
	"model_compliance_error": &XRError{
		Code:  400,
		Title: `The model provided would cause one or more entities in the Registry to become non-compliant.`,
	},
	"model_error": &XRError{
		Code:  400,
		Title: `There was an error in the model definition provided: <error_detail>.`,
	},
	"model_required_true": &XRError{
		Code:  400,
		Title: `Model attribute "<name>" needs to have a "required" value of "true" since a default value is provided.`,
	},
	"model_scalar_default": &XRError{
		Code:  400,
		Title: `Model attribute "<name>" is not allowed to have a default value since it is not a scalar.`,
	},
	"multiple_roots": &XRError{
		Code:  400,
		Title: `The operation would result in multiple root Versions for "<subject>", which is not allowed for "<plural>".`,
	},
	"not_found": &XRError{
		Code:  404,
		Title: `The targeted entity (<subject>) cannot be found.`,
	},
	"one_resource": &XRError{
		Code:  400,
		Title: `Only one attribute from "<list>" can be present at a time for: <subject>.`,
	},
	"parsing_data": &XRError{
		Code:  400,
		Title: `There was an error parsing the data: <error_detail>.`,
	},
	"readonly": &XRError{
		Code:  400,
		Title: `Updating a read-only entity (<subject>) is not allowed.`,
	},
	"required_attribute_missing": &XRError{
		Code:  400,
		Title: `One or more mandatory attributes for "<subject>" are missing: <list>.`,
	},
	"server_error": &XRError{
		Code:  500,
		Title: `An unexpected error occurred, please try again later.`,
	},
	"setdefaultversionid_not_allowed": &XRError{
		Code:  400,
		Title: `Processing "<subject>", the "setdefaultversionid" flag is not allowed to be specified for entities of type "<singular>".`,
	},
	"setdefaultversionsticky_false": &XRError{
		Code:  400,
		Title: `The model attribute "setdefaultversionsticky" needs to be "false" since "maxversions" is "1".`,
	},
	"sort_noncollection": &XRError{
		Code:  400,
		Title: `Can't sort on a non-collection result set. Query path: <subject>.`,
	},
	"too_large": &XRError{
		Code:  406,
		Title: `The size of the response is too large to return in a single response.`,
	},
	"too_many_versions": &XRError{
		Code:  400,
		Title: `When the "setdefaultversionid" flag is set to "request", only one Version is allowed to be specified in the request message.`,
	},
	"unknown_attribute": &XRError{
		Code:  400,
		Title: `An unknown attribute (<name>) was specified for "<subject>".`,
	},
	"unknown_id": &XRError{
		Code:  400,
		Title: `While processing "<subject>", the "<singular>" with a "<singular>id" value of "<id>" cannot be found.`,
	},
	"unsupported_specversion": &XRError{
		Code:  400,
		Title: `The specified "specversion" value (<specversion>) is not supported. Supported versions: <list>.`,
	},
	"versionid_not_allowed": &XRError{
		Code:  400,
		Title: `While creating a new Version for "<subject>", a "versionid" was specified but the "setversionid" model aspect for entities of type "<plural>" is "false".`,
	},
	"wrong_defaultversionid": &XRError{
		Code:  400,
		Title: `For "<subject>", the "defaultversionid" needs to be "<id>" since "defaultversionsticky" is "false".`,
	},

	// MODEL SPEC

	// HTTP SPEC
	"api_not_found": &XRError{
		Code:  404,
		Title: `The specified API is not supported: <subject>.`,
	},
	"details_required": &XRError{
		Code:  405,
		Title: `$details suffix is needed when using PATCH for entity: <subject>.`,
	},
	"extra_xregistry_header": &XRError{
		Code:  400,
		Title: `xRegistry HTTP header "<name>" is not allowed on this request: <error_detail>.`,
	},
	"header_error": &XRError{
		Code:  400,
		Title: `There was an error processing HTTP header "<name>": <error_detail>.`,
	},
	"missing_body": &XRError{
		Code:  400,
		Title: `The request is missing an HTTP body - try '{}'.`,
	},

	// CLIENT
	"exists": &XRError{
		Code:  400,
		Title: `"<subject>" already exists.`,
	},
	"client_error": &XRError{
		// Generic error
		Code:  400,
		Title: `<error_detail>.`,
	},
	"parsing_response": &XRError{
		Code:  400,
		Title: `There was an error parsing the response from the server: <error_detail>.`,
	},
	"talking_to_server": &XRError{
		Code:  400,
		Title: `There was an error talking to the server (<subject>): <error_detail>.`,
	},
}

func init() {
	for k, _ := range Type2Error {
		Type2Error[k].Type = SPECURL + "#" + k
	}
}

func (xErr *XRError) SetSubject(s string) *XRError {
	if xErr != nil {
		xErr.Subject = s
	}
	return xErr
}

func (xErr *XRError) SetTitle(t string) *XRError {
	if xErr != nil {
		xErr.Title = t
	}
	return xErr
}

var argNameRE = regexp.MustCompile("^[a-z][a-z_0-9]{0,62}$")
var argSpotRE = regexp.MustCompile("<[a-z][a-z_0-9]{0,62}>")

func (xErr *XRError) SetArgs(args ...string) *XRError {
	if xErr == nil {
		return nil
	}

	argMap := map[string]string{}
	for _, arg := range args {
		name, value, found := strings.Cut(arg, "=")
		PanicIf(!found, "Arg value %q is missing a \"=\".\nTitle: %s", arg,
			xErr.Title)
		PanicIf(!argNameRE.MatchString(name), "Arg name %q isn't valid", name)
		PanicIf(!strings.Contains(xErr.Title, "<"+name+">"),
			"Arg %q isn't in title ("+xErr.Type+")", name)

		argMap[name] = value
	}

	for _, name := range argSpotRE.FindAllString(xErr.Title, -1) {
		name = name[1 : len(name)-1] // remove <>'s
		if name == "subject" {
			PanicIf(xErr.Subject == "", "Subject is used but not set")
		} else {
			_, ok := argMap[name]
			PanicIf(!ok, "%q is in Title but not provided as an arg", name)
		}
	}

	xErr.Args = argMap

	return xErr
}

func (xErr *XRError) SetDetail(msg string) *XRError {
	if xErr != nil {
		xErr.Detail = msg
	}
	return xErr
}

func (xErr *XRError) SetDetailf(msg string, args ...any) *XRError {
	if xErr != nil {
		xErr.Detail = fmt.Sprintf(msg, args...)
	}
	return xErr
}

func (xErr *XRError) SetCode(c int) *XRError {
	if xErr != nil {
		xErr.Code = c
	}
	return xErr
}

func (xErr *XRError) SetHeader(name, value string) *XRError {
	if value == "null" {
		if xErr.Headers != nil {
			delete(xErr.Headers, name)
		}
		if len(xErr.Headers) == 0 {
			xErr.Headers = nil
		}
	} else {
		if xErr.Headers == nil {
			xErr.Headers = map[string]string{}
		}
		xErr.Headers[name] = value
	}
	return xErr
}

func (xErr *XRError) String() string {
	return xErr.ToJSON("")
}

func (xErr *XRError) ToJSON(baseURL string) string {
	type userErr struct {
		Type     string            `json:"type,omitempty"`
		Title    string            `json:"title,omitempty"`
		Detail   string            `json:"detail,omitempty"`
		Subject  string            `json:"subject,omitempty"`
		Args     map[string]string `json:"args,omitempty"`
		Instance string            `json:"instance,omitempty"`
		Source   string            `json:"source,omitempty"`
	}

	sub := xErr.Subject
	if len(sub) == 0 {
		// sub = "/"
	} else if sub[0] != '/' && !strings.HasPrefix(sub, "http") {
		// If we ever change e.Path so it starts with "/" then we should
		// be able to stop looking for "http" as a special case
		sub = "/" + sub
	}
	// sub = strings.TrimRight(baseURL, "/") + sub

	tmpErr := userErr{
		Type:     xErr.Type,
		Title:    xErr.GetTitle(),
		Detail:   xErr.Detail,
		Subject:  sub,
		Args:     xErr.Args,
		Instance: xErr.Instance,
		Source:   xErr.Source,
	}

	// can't use MarshalIndent because go escapes chars, like "<"  :-(
	// buf, _ := json.MarshalIndent(tmpErr, "", "  ")
	buf := bytes.Buffer{}
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	encoder.Encode(tmpErr)

	// Why Encode() add a \n when Marshal() doesn't is beyond me!
	res := strings.TrimRight(buf.String(), "\n")

	return res
}

func (xErr *XRError) GetTitle() string {
	fn := func(arg string) string {
		arg = arg[1 : len(arg)-1] // remove <>'s
		if arg == "subject" {
			return xErr.Subject
		}
		return xErr.Args[arg]
	}
	title := argSpotRE.ReplaceAllStringFunc(xErr.Title, fn)
	return title
}

func (xErr *XRError) IsType(daType string) bool {
	if xErr == nil {
		return false
	}
	_, t, found := strings.Cut(xErr.Type, "#")
	PanicIf(!found, "No # found in: %s", xErr)
	return t == daType
}

func ParseXRError(buf []byte) (*XRError, error) {
	xErr := XRError{}
	if err := json.Unmarshal(buf, &xErr); err != nil {
		return nil, err
	}
	return &xErr, nil
}

func (xErr *XRError) AddSource(src string) *XRError {
	if src == "" {
		pc, file, lineNum, ok := runtime.Caller(1)
		if ok {
			pkg, _, _ := strings.Cut(path.Base(runtime.FuncForPC(pc).Name()), ".")
			file, _, _ = strings.Cut(path.Base(file), ".")
			if xErr.Source == "" {
				xErr.Source = fmt.Sprintf("%.12s:", GitCommit)
			}
			src = fmt.Sprintf("%s:%s:%d", pkg, file, lineNum)
		} else {
			src = "<na>"
		}
	}

	if xErr.Source == "" {
		xErr.Source = src
	} else {
		xErr.Source += "," + src
	}
	return xErr
}

// Add the caller's parent func to the Source.
// Used by popular util func to add parent to the Source's stack trace
func (xErr *XRError) AddSourceParent() *XRError {
	src := "<na>"
	pc, file, lineNum, ok := runtime.Caller(2)
	if ok {
		pkg, _, _ := strings.Cut(path.Base(runtime.FuncForPC(pc).Name()), ".")
		file, _, _ = strings.Cut(path.Base(file), ".")
		if xErr.Source == "" {
			xErr.Source = fmt.Sprintf("%.12s:", GitCommit)
		}
		src = fmt.Sprintf("%s:%s:%d", pkg, file, lineNum)
	}
	return xErr.AddSource(src)
}
