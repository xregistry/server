package common

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	log "github.com/duglin/dlog"
)

type XRError struct {
	Type      string
	Title     string
	TitleArgs []any
	Instance  string
	Detail    string
	Code      int               // HTTP response code
	Headers   map[string]string // HTTP headers to include in response
}

func (xErr *XRError) Error() string {
	return xErr.ToUserJson("")
	/*
		str := fmt.Sprintf(xErr.Title, xErr.TitleArgs...)
		if xErr.Detail != "" {
			str += " (" + xErr.Detail + ")"
		}
		return str
	*/
}

func NewXRErrorWithTitle(code int, title string) *XRError {
	return &XRError{
		Code:  code,
		Title: title,
	}
}

func NewXRError(daType string, instance string, args ...any) *XRError {
	err := Type2Error[daType]
	PanicIf(err == nil, "Unknown error type: %s", daType)

	if daType == "server_error" {
		// This is special.
		// Only show internal info (args) to the logs.
		// Remove internal stuff from the error that's bubbled up.
		if len(args) > 0 {
			log.Print(args...)
			args = nil
		}
		log.Printf(">>> System Error <<<")
		ShowStack()
	}
	// ShowStack()

	return &XRError{
		Type:      err.Type,
		Instance:  instance,
		Title:     err.Title,
		TitleArgs: args,
		Detail:    err.Detail,
		Code:      err.Code,
		Headers:   maps.Clone(err.Headers),
	}
}

// If not already an XRError, convert the "error" to one using daType
func AsXRError(err error, daType string, instance string) *XRError {
	xErr, ok := err.(*XRError)
	if !ok {
		xErr = NewXRError(daType, instance)
		xErr.SetDetail(err.Error())
	}
	return xErr
}

func (xErr *XRError) AsError() error {
	return (error)(xErr)
}

var Type2Error = map[string]*XRError{
	// CODE SPEC
	"action_not_supported": &XRError{
		Code:  405,
		Title: `The specified action (%s) is not supported`,
	},
	"ancestor_circular_reference": &XRError{
		Code:  400,
		Title: `The assigned "ancestor" value (%s) creates a circular reference`,
	},
	"bad_flag": &XRError{
		Code:  400,
		Title: `The specified flag (%s) is not allowed in this context`,
	},
	"bad_request": &XRError{
		Code:  400,
		Title: `The request cannot be processed as provided: %s`,
	},
	"cannot_doc_xref": &XRError{
		Code:  400,
		Title: `Retrieving the document view of a Version whose Resource uses "xref" is not possible`,
	},
	"capability_error": &XRError{
		Code:  400,
		Title: `There was an error in the capabilities provided: %s`,
	},
	"compatibility_violation": &XRError{
		Code:  400,
		Title: `The request would cause one or more Versions of this Resource to violate the Resource's compatibility rule (%s)`,
	},
	"data_retrieval_error": &XRError{
		Code:  500,
		Title: `The server was unable to retrieve all of the requested data: %s`,
	},
	"defaultversionid_not_allowed": &XRError{
		Code:  400,
		Title: `"defaultversionid" is not allowed to be specified for Resources of type %q`,
	},
	"invalid_attribute": &XRError{
		Code:  400,
		Title: `The attribute %q is not valid: %s`,
	},
	"invalid_data": &XRError{
		Code:  400,
		Title: `The data provided for "%s" is invalid`,
	},
	"mismatched_epoch": &XRError{
		Code:  400,
		Title: `The specified epoch value (%v) does not match its current value (%v)`,
	},
	"mismatched_id": &XRError{
		Code:  400,
		Title: `The specified "%s" ID value (%s) needs to be "%s"`,
	},
	"misplaced_epoch": &XRError{
		Code:  400,
		Title: `The specified "epoch" value needs to be within a "meta" entity`,
	},
	"missing_versions": &XRError{
		Code:  400,
		Title: `At least one Version needs to be included in the request`,
	},
	"model_compliance_error": &XRError{
		Code:  400,
		Title: `The model provided would cause one or more entities in the Registry to become non-compliant`,
	},
	"model_error": &XRError{
		Code:  400,
		Title: `There was an error in the model definition provided: %s`,
	},
	"multiple_roots": &XRError{
		Code:  400,
		Title: `The operation would result in multiple root Versions which is not allowed by this Resource type`,
	},
	"not_found": &XRError{
		Code:  404,
		Title: `The specified entity cannot be found: %s`,
	},
	"readonly": &XRError{
		Code:  400,
		Title: `Updating a read-only entity is not allowed: %s`,
	},
	"required_attribute_missing": &XRError{
		Code:  400,
		Title: `One or more mandatory attributes are missing: %s`,
	},
	"server_error": &XRError{
		Code:  500,
		Title: `An unexpected error occurred, please try again later`,
	},
	"too_large": &XRError{
		Code:  400,
		Title: `The size of the response is too large to return in a single response`,
	},
	"too_many_versions": &XRError{
		Code:  400,
		Title: `The request is only allowed to have one Version specified`,
	},
	"unknown_attribute": &XRError{
		Code:  400,
		Title: `An unknown attribute (%s) was specified`,
	},
	"unknown_id": &XRError{
		Code:  400,
		Title: `The "%s" with the ID "%s" cannot be found`,
	},
	"unsupported_specversion": &XRError{
		Code:  400,
		Title: `The specified "specversion" value (%s) is not supported`,
	},
	"versionid_not_allowed": &XRError{
		Code:  400,
		Title: `A "versionid" (%s) is not allowed to be specified`,
	},

	// MODEL SPEC

	// HTTP SPEC
	"api_not_found": &XRError{
		Code:  404,
		Title: `The specified API is not supported: %s`,
	},
	"details_required": &XRError{
		Code:  405,
		Title: `$details suffixed is needed when using PATCH for this Resource`,
	},
	"extra_xregistry_headers": &XRError{
		Code:  400,
		Title: `xRegistry HTTP headers are not allowed on this request`,
	},
	"header_decoding_error": &XRError{
		Code:  400,
		Title: `The value ("%s") of the HTTP "%s" header cannot be decoded`,
	},
	"missing_body": &XRError{
		Code:  400,
		Title: `The request is missing an HTTP body - try '{}'`,
	},
}

func init() {
	for k, _ := range Type2Error {
		Type2Error[k].Type = SPECURL + "#" + k
	}
}

func (xErr *XRError) SetInstance(i string) *XRError {
	if xErr != nil {
		xErr.Instance = i
	}
	return xErr
}

func (xErr *XRError) SetTitle(t string) *XRError {
	if xErr != nil {
		xErr.Title = t
	}
	return xErr
}

func (xErr *XRError) SetTitleArgs(args ...any) *XRError {
	if xErr != nil {
		xErr.TitleArgs = args
	}
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
	return xErr.ToUserJson("")
	str := fmt.Sprintf(xErr.Title, xErr.TitleArgs)
	str = fmt.Sprintf("%d/%s: %s", xErr.Code, xErr.Type, str)

	if xErr.Instance != "" {
		str += "(" + xErr.Instance + ")"
	}

	if xErr.Detail != "" {
		str += "\n" + xErr.Detail
	}

	if len(xErr.Headers) != 0 {
		str += "\n" + fmt.Sprintf("%v", xErr.Headers)
	}

	return str
}

func (xErr *XRError) ToUserJson(baseURL string) string {
	str := "{\n"
	str += fmt.Sprintf("  \"type\": %q,\n", xErr.Type)
	inst := xErr.Instance
	if len(inst) == 0 {
		inst = "/"
	} else if inst[0] != '/' {
		inst = "/" + inst
	}

	u := strings.TrimRight(baseURL, "/")
	str += fmt.Sprintf("  \"instance\": %q,\n", u+inst)
	str += fmt.Sprintf("  \"title\": %q", xErr.GetTitle())
	if xErr.Detail != "" {
		str += fmt.Sprintf(",\n  \"detail\": %q", xErr.Detail)
	}
	str += "\n}"
	return str
}

func (xErr *XRError) GetTitle() string {
	return fmt.Sprintf(xErr.Title, xErr.TitleArgs...)
}

func ParseXRError(buf []byte) (*XRError, error) {
	xErr := XRError{}
	if err := json.Unmarshal(buf, &xErr); err != nil {
		return nil, err
	}
	return &xErr, nil
}

/*
func SetXRErrorInstance(err error, xid string) *XRError {
	if err == nil {
		return nil
	}
	xErr, ok := err.(*XRError)
	if !ok {
		return err
	}
	xErr.SetInstance(xid)
	return xErr
}
*/
