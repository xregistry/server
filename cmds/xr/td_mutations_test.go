package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestMutationsRequiresAuthorization(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			requests++
			http.Error(w, "unexpected request", http.StatusInternalServerError)
		}))
	defer server.Close()

	root := newConformMutationTestTD(server.URL)
	result := root.Run(TestMutations)

	XEqual(t, "Status", result.Status, SKIP)
	XEqual(t, "Requests", requests, 0)
}

func TestConformMutationGroupModel(t *testing.T) {
	tests := []struct {
		name string
		attr *registry.Attribute
		key  string
		safe bool
	}{
		{
			name: "optional extension",
			key:  "optional",
			attr: &registry.Attribute{Type: STRING},
			safe: true,
		},
		{
			name: "readonly required extension",
			key:  "servervalue",
			attr: &registry.Attribute{
				Type:     STRING,
				ReadOnly: true,
				Required: true,
			},
		},
		{
			name: "client required extension without default",
			key:  "required",
			attr: &registry.Attribute{
				Type:     STRING,
				Required: true,
			},
		},
		{
			name: "client required extension with string default",
			key:  "required",
			attr: &registry.Attribute{
				Type:     STRING,
				Required: true,
				Default:  "default",
			},
			safe: true,
		},
		{
			name: "client required extension with empty string default",
			key:  "required",
			attr: &registry.Attribute{
				Type:     STRING,
				Required: true,
				Default:  "",
			},
			safe: true,
		},
		{
			name: "client required extension with false default",
			key:  "required",
			attr: &registry.Attribute{
				Type:     BOOLEAN,
				Required: true,
				Default:  false,
			},
			safe: true,
		},
		{
			name: "client required extension with zero default",
			key:  "required",
			attr: &registry.Attribute{
				Type:     UINTEGER,
				Required: true,
				Default:  0,
			},
			safe: true,
		},
		{
			name: "required core name without default",
			key:  "name",
			attr: &registry.Attribute{
				Type:     STRING,
				Required: true,
			},
		},
		{
			name: "required server-populated id",
			key:  "dirid",
			attr: &registry.Attribute{
				Type:      STRING,
				Immutable: true,
				Required:  true,
			},
			safe: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := &registry.Model{
				Groups: map[string]*registry.GroupModel{
					"dirs": {
						Plural:   "dirs",
						Singular: "dir",
						Attributes: registry.Attributes{
							test.key: test.attr,
						},
					},
				},
			}
			_, groupModel, reasons := conformMutationGroupModel(model)
			XEqual(t, "Safe", groupModel != nil, test.safe)
			if !test.safe {
				XCheck(t, len(reasons) == 1,
					"Expected one deterministic rejection reason: %#v", reasons)
			}
		})
	}

	_, groupModel, reasons := conformMutationGroupModel(&registry.Model{})
	XCheck(t, groupModel == nil, "Empty model MUST be unsuitable")
	XEqual(t, "Empty model reason", reasons,
		[]string{"The model defines no Group Types."})
}

func TestMutationsCleanup(t *testing.T) {
	tests := []struct {
		name             string
		failFast         bool
		withoutEpochFlag bool
		epochZero        bool
		unsafeEpoch      uint64
		preflightExists  bool
		wrongLocation    bool
		unconfirmed      bool
		foreignResource  bool
		unexpectedStatus bool
		redirectCode     int
		epochChanged     bool
		wantStatus       int
		wantMethods      string
		wantDeletes      int
		wantRetained     bool
	}{
		{
			name:        "success",
			wantStatus:  PASS,
			wantMethods: "GET,PUT,GET,GET,DELETE,GET",
			wantDeletes: 1,
		},
		{
			name:        "success with epoch zero",
			epochZero:   true,
			wantStatus:  PASS,
			wantMethods: "GET,PUT,GET,GET,DELETE,GET",
			wantDeletes: 1,
		},
		{
			name:         "success through 307 redirect",
			redirectCode: http.StatusTemporaryRedirect,
			wantStatus:   PASS,
			wantMethods:  "GET,PUT,GET,GET,DELETE,GET",
			wantDeletes:  1,
		},
		{
			name:         "success through 308 redirect",
			redirectCode: http.StatusPermanentRedirect,
			wantStatus:   PASS,
			wantMethods:  "GET,PUT,GET,GET,DELETE,GET",
			wantDeletes:  1,
		},
		{
			name:             "success without epoch flag",
			withoutEpochFlag: true,
			wantStatus:       PASS,
			wantMethods:      "GET,PUT,GET,GET,DELETE,GET",
			wantDeletes:      1,
		},
		{
			name:            "preflight object retained",
			preflightExists: true,
			wantStatus:      FAIL,
			wantMethods:     "GET",
			wantRetained:    true,
		},
		{
			name:          "stop cleanup",
			wrongLocation: true,
			wantStatus:    FAIL,
			wantMethods:   "GET,PUT,GET,DELETE,GET",
			wantDeletes:   1,
		},
		{
			name:          "failfast cleanup",
			failFast:      true,
			wrongLocation: true,
			wantStatus:    FAIL,
			wantMethods:   "GET,PUT,GET,DELETE,GET",
			wantDeletes:   1,
		},
		{
			name:         "unconfirmed object retained",
			unconfirmed:  true,
			wantStatus:   FAIL,
			wantMethods:  "GET,PUT,GET",
			wantDeletes:  0,
			wantRetained: true,
		},
		{
			name:            "foreign nested resource retained",
			foreignResource: true,
			wantStatus:      FAIL,
			wantMethods:     "GET,PUT,GET,GET",
			wantDeletes:     0,
			wantRetained:    true,
		},
		{
			name:             "unexpected success status",
			unexpectedStatus: true,
			wantStatus:       FAIL,
			wantMethods:      "GET,PUT,GET",
			wantDeletes:      0,
			wantRetained:     true,
		},
		{
			name:             "unexpected success status failfast",
			failFast:         true,
			unexpectedStatus: true,
			wantStatus:       FAIL,
			wantMethods:      "GET,PUT,GET",
			wantDeletes:      0,
			wantRetained:     true,
		},
		{
			name:         "epoch change retained",
			epochChanged: true,
			wantStatus:   FAIL,
			wantMethods:  "GET,PUT,GET,GET",
			wantDeletes:  0,
			wantRetained: true,
		},
		{
			name:         "unsafe epoch response retained",
			unsafeEpoch:  9007199254740993,
			wantStatus:   FAIL,
			wantMethods:  "GET,PUT,GET",
			wantDeletes:  0,
			wantRetained: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var mu sync.Mutex
			methods := []string{}
			deletes := 0
			current := map[string]any(nil)
			ownedGets := 0
			ownedID := ""
			redirects := 0
			publicURL := ""

			backend := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					methods = append(methods, r.Method)

					w.Header().Set("Content-Type", "application/json")
					switch r.Method {
					case http.MethodGet:
						if test.preflightExists {
							current = map[string]any{"name": "foreign"}
						}
						if current == nil {
							w.WriteHeader(http.StatusNotFound)
							_, _ = w.Write([]byte(`{"title":"not found"}`))
							return
						}
						_ = json.NewEncoder(w).Encode(current)
						ownedGets++
						if test.epochChanged && ownedGets == 1 {
							current["epoch"] = 2
						}
					case http.MethodPut:
						body := map[string]any{}
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Errorf("decode request: %v", err)
						}
						if body["epoch"] != conformMutationEpochSentinel {
							t.Errorf("epoch sentinel: %#v", body["epoch"])
						}

						id := strings.TrimPrefix(r.URL.Path, "/dirs/")
						ownedID = id
						if body["dirid"] != id {
							t.Errorf("dirid: %#v, want %q", body["dirid"], id)
						}

						if test.unconfirmed {
							current = map[string]any{
								"dirid":      id,
								"xid":        r.URL.Path,
								"epoch":      7,
								"filescount": 0,
							}
							w.WriteHeader(http.StatusInternalServerError)
							_, _ = w.Write([]byte(`{"type":"test","title":"failed"}`))
							return
						}

						epoch := any(1)
						if test.epochZero {
							epoch = 0
						}
						if test.unsafeEpoch != 0 {
							epoch = test.unsafeEpoch
						}
						current = map[string]any{
							"dirid":      id,
							"xid":        r.URL.Path,
							"epoch":      epoch,
							"filescount": 0,
						}
						location := publicURL + r.URL.Path
						if test.wrongLocation {
							location += "-wrong"
						}
						w.Header().Set("Location", location)
						status := http.StatusCreated
						if test.unexpectedStatus {
							status = http.StatusOK
						}
						w.WriteHeader(status)
						_ = json.NewEncoder(w).Encode(current)
						if test.foreignResource {
							current = map[string]any{
								"dirid":      id,
								"xid":        r.URL.Path,
								"epoch":      2,
								"filescount": 1,
								"files": map[string]any{
									"foreign": map[string]any{},
								},
							}
						}
					case http.MethodDelete:
						deletes++
						expectedEpoch := fmt.Sprint(current["epoch"])
						if test.withoutEpochFlag {
							if r.URL.Path != "/dirs" {
								t.Errorf("delete path: %q, want /dirs", r.URL.Path)
							}
							if r.URL.RawQuery != "" {
								t.Errorf("delete query: %q, want empty", r.URL.RawQuery)
							}
							deleteBody := map[string]map[string]json.Number{}
							decoder := json.NewDecoder(r.Body)
							decoder.UseNumber()
							if err := decoder.Decode(&deleteBody); err != nil {
								t.Errorf("decode delete body: %v", err)
							}
							if len(deleteBody) != 1 {
								t.Errorf("delete body entries: %d, want 1",
									len(deleteBody))
							}
							entry, ok := deleteBody[ownedID]
							if !ok {
								t.Errorf("delete body missing %q", ownedID)
							} else if fmt.Sprint(entry["epoch"]) != expectedEpoch {
								t.Errorf("delete body epoch: %q, want %q",
									entry["epoch"], expectedEpoch)
							}
						} else {
							if r.URL.Path != "/dirs/"+ownedID {
								t.Errorf("delete path: %q, want /dirs/%s",
									r.URL.Path, ownedID)
							}
							if r.URL.Query().Get("epoch") != expectedEpoch {
								t.Errorf("delete epoch: %q",
									r.URL.Query().Get("epoch"))
							}
						}
						current = nil
						w.WriteHeader(http.StatusNoContent)
					default:
						http.Error(w, "unexpected method",
							http.StatusMethodNotAllowed)
					}
				}))
			defer backend.Close()

			targetURL := backend.URL
			publicURL = backend.URL
			if test.redirectCode != 0 {
				redirect := httptest.NewServer(http.HandlerFunc(
					func(w http.ResponseWriter, r *http.Request) {
						mu.Lock()
						redirects++
						mu.Unlock()
						w.Header().Set("Location",
							backend.URL+r.URL.RequestURI())
						w.WriteHeader(test.redirectCode)
					}))
				defer redirect.Close()
				targetURL = redirect.URL
				publicURL = redirect.URL
			}

			root := newConformMutationTestTD(targetURL)
			root.Config.AllowMutations = true
			root.Config.FailFast = test.failFast
			if test.withoutEpochFlag {
				root.GetRegistry().Capabilities.Flags = nil
			}
			result := root.Run(TestMutations)

			mu.Lock()
			gotMethods := strings.Join(methods, ",")
			gotDeletes := deletes
			gotRedirects := redirects
			mu.Unlock()
			XEqual(t, "Status", result.Status, test.wantStatus)
			XEqual(t, "Methods", gotMethods, test.wantMethods)
			XEqual(t, "Deletes", gotDeletes, test.wantDeletes)
			if test.redirectCode != 0 {
				XEqual(t, "Redirected requests", gotRedirects,
					len(methods))
			}
			if test.wantRetained {
				XCheck(t, current != nil, "Object must remain")
			} else {
				XCheck(t, current == nil, "Run-owned object must be removed")
			}
		})
	}
}

func TestConformMutationUInt(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		want    uint64
		wantErr bool
	}{
		{name: "zero", value: float64(0), want: 0},
		{
			name:  "largest exact JSON integer",
			value: float64(9007199254740991),
			want:  9007199254740991,
		},
		{
			name:    "two to the fifty-third",
			value:   float64(9007199254740992),
			wantErr: true,
		},
		{name: "fraction", value: 1.5, wantErr: true},
		{name: "negative", value: float64(-1), wantErr: true},
		{name: "wrong type", value: "1", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := conformMutationUInt(test.value)
			XEqual(t, "Error", err != nil, test.wantErr)
			if !test.wantErr {
				XEqual(t, "Value", got, test.want)
			}
		})
	}
}

type mutationRunServer struct {
	server     *httptest.Server
	mu         sync.Mutex
	requests   int
	writes     int
	createdIDs []string
	current    map[string]any
	failCreate bool
}

func newMutationRunServer(t *testing.T, failCreate bool) *mutationRunServer {
	t.Helper()

	recorder := &mutationRunServer{failCreate: failCreate}
	recorder.server = httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			recorder.mu.Lock()
			defer recorder.mu.Unlock()
			recorder.requests++
			w.Header().Set("Content-Type", "application/json")

			switch r.URL.Path {
			case "/model":
				_, _ = w.Write([]byte(`{
  "groups": {
    "dirs": {
      "singular": "dir",
      "resources": {
        "files": {"singular": "file"}
      }
    }
  }
}`))
				return
			case "/capabilities":
				_, _ = w.Write([]byte(`{
  "available": {
    "capabilities": {"mutable": true},
    "entities": {"mutable": true},
    "model": {"mutable": false}
  },
  "flags": ["epoch"]
}`))
				return
			case "/":
				_, _ = w.Write([]byte(`{}`))
				return
			}

			switch r.Method {
			case http.MethodGet:
				if recorder.current == nil {
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"title":"not found"}`))
					return
				}
				_ = json.NewEncoder(w).Encode(recorder.current)
			case http.MethodPut:
				recorder.writes++
				id := strings.TrimPrefix(r.URL.Path, "/dirs/")
				recorder.createdIDs = append(recorder.createdIDs, id)
				if recorder.failCreate {
					http.Error(w, "create failed",
						http.StatusInternalServerError)
					return
				}
				recorder.current = map[string]any{
					"dirid":      id,
					"xid":        r.URL.Path,
					"epoch":      1,
					"filescount": 0,
				}
				w.Header().Set("Location", recorder.server.URL+r.URL.Path)
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(recorder.current)
			case http.MethodDelete:
				recorder.writes++
				recorder.current = nil
				w.WriteHeader(http.StatusNoContent)
			default:
				http.Error(w, "unexpected method",
					http.StatusMethodNotAllowed)
			}
		}))
	return recorder
}

func (recorder *mutationRunServer) close() {
	recorder.server.Close()
}

func TestMutationsMultipleTargets(t *testing.T) {
	first := newMutationRunServer(t, false)
	defer first.close()
	second := newMutationRunServer(t, false)
	defer second.close()

	config := &TDConfig{
		AllowMutations: true,
		ConsoleDepth:   1,
		IgnoreWarn:     true,
		Out:            io.Discard,
		Output:         "text",
		RunFuncs:       []TestFn{TestMutations},
	}
	rc := runConform([]string{first.server.URL, second.server.URL}, config)
	XEqual(t, "Exit code", rc, 0)

	first.mu.Lock()
	firstWrites := first.writes
	firstID := first.createdIDs[0]
	firstCurrent := first.current
	first.mu.Unlock()
	second.mu.Lock()
	secondWrites := second.writes
	secondID := second.createdIDs[0]
	secondCurrent := second.current
	second.mu.Unlock()

	XEqual(t, "First target writes", firstWrites, 2)
	XEqual(t, "Second target writes", secondWrites, 2)
	XCheck(t, firstID != secondID,
		"Each target MUST receive an independent mutation ID")
	XCheck(t, firstCurrent == nil, "First target MUST be cleaned up")
	XCheck(t, secondCurrent == nil, "Second target MUST be cleaned up")
}

func TestMutationsMultipleTargetsFailFast(t *testing.T) {
	first := newMutationRunServer(t, true)
	defer first.close()
	second := newMutationRunServer(t, false)
	defer second.close()

	config := &TDConfig{
		AllowMutations: true,
		ConsoleDepth:   1,
		FailFast:       true,
		IgnoreWarn:     true,
		Out:            io.Discard,
		Output:         "text",
		RunFuncs:       []TestFn{TestMutations},
	}
	rc := runConform([]string{first.server.URL, second.server.URL}, config)
	XEqual(t, "Exit code", rc, 1)

	second.mu.Lock()
	secondRequests := second.requests
	second.mu.Unlock()
	XEqual(t, "Second target requests", secondRequests, 0)
}

func newConformMutationTestTD(serverURL string) *TD {
	root := newTestTD("Mutation")
	reg, xErr := registry.DefineRegistry(serverURL, XRConfig)
	root.NoErrorStop(xErr)
	reg.Model = &registry.Model{
		Groups: map[string]*registry.GroupModel{
			"dirs": {
				Plural:     "dirs",
				Singular:   "dir",
				Attributes: registry.Attributes{},
				Resources: map[string]*registry.ResourceModel{
					"files": {
						Plural:   "files",
						Singular: "file",
					},
				},
			},
		},
	}
	reg.Capabilities = &Capabilities{
		Available: map[string]*AvailableObject{
			"entities": {Mutable: true},
		},
		Flags: []string{"epoch"},
	}
	root.SetRegistry(reg)
	cachePassedTest(root, TestModel)
	cachePassedTest(root, TestCapabilities)
	return root
}
