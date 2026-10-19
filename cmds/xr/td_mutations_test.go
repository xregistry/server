package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	. "github.com/xregistry/server/common"
	"github.com/xregistry/server/registry"
)

func TestTDGroupCreateReadDeleteRequiresAuthorization(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			requests++
			http.Error(w, "unexpected request", http.StatusInternalServerError)
		}))
	defer server.Close()

	root := newConformMutationTestTD(server.URL)
	result := root.Run(TestTDGroupCreateReadDelete)

	XEqual(t, "Status", result.Status, FAIL)
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
			safe: true,
		},
		{
			name: "client required extension",
			key:  "required",
			attr: &registry.Attribute{
				Type:     STRING,
				Required: true,
				Default:  "default",
			},
		},
		{
			name: "conditional readonly extension",
			key:  "conditional",
			attr: &registry.Attribute{
				Type:     STRING,
				ReadOnly: true,
				Required: true,
				IfValues: registry.IfValues{
					"value": {},
				},
			},
		},
		{
			name: "core labels",
			key:  "labels",
			attr: &registry.Attribute{
				Type: MAP,
				Item: &registry.Item{Type: STRING},
			},
			safe: true,
		},
		{
			name: "readonly labels",
			key:  "labels",
			attr: &registry.Attribute{
				Type:     MAP,
				ReadOnly: true,
				Item:     &registry.Item{Type: STRING},
			},
		},
		{
			name: "immutable labels",
			key:  "labels",
			attr: &registry.Attribute{
				Type:      MAP,
				Immutable: true,
				Item:      &registry.Item{Type: STRING},
			},
		},
		{
			name: "constrained label values",
			key:  "labels",
			attr: &registry.Attribute{
				Type: MAP,
				Item: &registry.Item{
					Type: STRING,
					Enum: []any{"allowed"},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			model := &registry.Model{
				Groups: map[string]*registry.GroupModel{
					"widgets": {
						Plural:   "widgets",
						Singular: "widget",
						Attributes: registry.Attributes{
							test.key: test.attr,
						},
					},
				},
			}
			_, groupModel := conformMutationGroupModel(model)
			XEqual(t, "Safe", groupModel != nil, test.safe)
		})
	}
}

func TestTDGroupCreateReadDeleteSkipsReadonlyLabels(t *testing.T) {
	var requests int
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			requests++
			http.Error(w, "unexpected request", http.StatusInternalServerError)
		}))
	defer server.Close()

	root := newConformMutationTestTD(server.URL)
	root.Config.AllowMutations = true
	root.GetRegistry().Model.Groups["widgets"].Attributes["labels"] =
		&registry.Attribute{
			Type:     MAP,
			ReadOnly: true,
			Item:     &registry.Item{Type: STRING},
		}
	result := root.Run(TestTDGroupCreateReadDelete)

	XEqual(t, "Status", result.Status, SKIP)
	XEqual(t, "Requests", requests, 0)
}

func TestTDGroupCreateReadDeleteCleanup(t *testing.T) {
	tests := []struct {
		name             string
		failFast         bool
		preflightExists  bool
		wrongLocation    bool
		foreignObject    bool
		foreignResource  bool
		unexpectedStatus bool
		createRedirect   bool
		deleteRedirect   bool
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
			name:          "foreign object retained",
			foreignObject: true,
			wantStatus:    FAIL,
			wantMethods:   "GET,PUT,GET",
			wantDeletes:   0,
			wantRetained:  true,
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
			name:           "create redirect rejected",
			createRedirect: true,
			wantStatus:     FAIL,
			wantMethods:    "GET,PUT,GET",
			wantDeletes:    0,
		},
		{
			name:           "delete redirect rejected",
			deleteRedirect: true,
			wantStatus:     FAIL,
			wantMethods:    "GET,PUT,GET,GET,DELETE",
			wantDeletes:    1,
			wantRetained:   true,
		},
		{
			name:         "large epoch change retained",
			epochChanged: true,
			wantStatus:   FAIL,
			wantMethods:  "GET,PUT,GET,GET",
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
			foreignTouched := false
			var server *httptest.Server

			server = httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					methods = append(methods, r.Method)
					if strings.HasSuffix(r.URL.Path, "/foreign") {
						foreignTouched = true
					}

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
							current["epoch"] = uint64(9007199254740996)
						}
					case http.MethodPut:
						if test.createRedirect {
							w.Header().Set("Location",
								server.URL+"/widgets/foreign")
							w.WriteHeader(http.StatusTemporaryRedirect)
							return
						}

						body := map[string]any{}
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Errorf("decode request: %v", err)
						}
						if body["epoch"] != conformMutationEpochSentinel {
							t.Errorf("epoch sentinel: %#v", body["epoch"])
						}

						id := strings.TrimPrefix(r.URL.Path, "/widgets/")
						if body["widgetid"] != id {
							t.Errorf("widgetid: %#v, want %q", body["widgetid"], id)
						}
						labels, _ := body["labels"].(map[string]any)
						owner, _ := labels[conformMutationLabel].(string)

						if test.foreignObject {
							current = map[string]any{
								"widgetid": id,
								"xid":      r.URL.Path,
								"epoch":    7,
								"labels": map[string]any{
									conformMutationLabel: "foreign",
								},
							}
							w.WriteHeader(http.StatusInternalServerError)
							_, _ = w.Write([]byte(`{"type":"test","title":"failed"}`))
							return
						}

						epoch := any(1)
						if test.epochChanged {
							epoch = uint64(9007199254740995)
						}
						current = map[string]any{
							"widgetid":    id,
							"xid":         r.URL.Path,
							"epoch":       epoch,
							"thingscount": 0,
							"labels": map[string]any{
								conformMutationLabel: owner,
							},
						}
						location := server.URL + r.URL.Path
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
								"widgetid":    id,
								"xid":         r.URL.Path,
								"epoch":       2,
								"thingscount": 1,
								"things": map[string]any{
									"foreign": map[string]any{},
								},
								"labels": map[string]any{
									conformMutationLabel: owner,
								},
							}
						}
					case http.MethodDelete:
						deletes++
						expectedEpoch := fmt.Sprint(current["epoch"])
						if r.URL.Query().Get("epoch") != expectedEpoch {
							t.Errorf("delete epoch: %q", r.URL.Query().Get("epoch"))
						}
						if test.deleteRedirect {
							w.Header().Set("Location",
								server.URL+"/widgets/foreign")
							w.WriteHeader(http.StatusTemporaryRedirect)
							return
						}
						current = nil
						w.WriteHeader(http.StatusNoContent)
					default:
						http.Error(w, "unexpected method",
							http.StatusMethodNotAllowed)
					}
				}))
			defer server.Close()

			root := newConformMutationTestTD(server.URL)
			root.Config.AllowMutations = true
			root.Config.FailFast = test.failFast
			result := root.Run(TestTDGroupCreateReadDelete)

			mu.Lock()
			gotMethods := strings.Join(methods, ",")
			gotDeletes := deletes
			gotForeignTouched := foreignTouched
			mu.Unlock()
			XEqual(t, "Status", result.Status, test.wantStatus)
			XEqual(t, "Methods", gotMethods, test.wantMethods)
			XEqual(t, "Deletes", gotDeletes, test.wantDeletes)
			XEqual(t, "Foreign redirect target", gotForeignTouched, false)
			if test.wantRetained {
				XCheck(t, current != nil, "Object must remain")
			} else {
				XCheck(t, current == nil, "Run-owned object must be removed")
			}
		})
	}
}

func newConformMutationTestTD(serverURL string) *TD {
	root := newTestTD("Mutation")
	reg, xErr := registry.DefineRegistry(serverURL, XRConfig)
	root.NoErrorStop(xErr)
	reg.Model = &registry.Model{
		Groups: map[string]*registry.GroupModel{
			"widgets": {
				Plural:     "widgets",
				Singular:   "widget",
				Attributes: registry.Attributes{},
				Resources: map[string]*registry.ResourceModel{
					"things": {
						Plural:   "things",
						Singular: "thing",
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
