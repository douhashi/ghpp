package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/shurcooL/githubv4"
)

// graphqlResponse is a helper to build JSON responses for the mock server.
type graphqlResponse struct {
	Data   interface{} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

func TestFetchProjectItems_User(t *testing.T) {
	resp := graphqlResponse{
		Data: map[string]interface{}{
			"user": map[string]interface{}{
				"projectV2": map[string]interface{}{
					"items": map[string]interface{}{
						"totalCount": 3,
						"pageInfo": map[string]interface{}{
							"hasNextPage": false,
							"endCursor":   "",
						},
						"nodes": []interface{}{
							map[string]interface{}{
								"id": "PVTI_001",
								"fieldValues": map[string]interface{}{
									"nodes": []interface{}{
										map[string]interface{}{
											"__typename": "ProjectV2ItemFieldTextValue",
										},
										map[string]interface{}{
											"__typename": "ProjectV2ItemFieldSingleSelectValue",
											"name":       "Ready",
											"field": map[string]interface{}{
												"__typename": "ProjectV2SingleSelectField",
												"name":       "Status",
											},
										},
									},
								},
								"content": map[string]interface{}{
									"__typename": "Issue",
									"title":      "Sample Issue 2",
									"url":        "https://github.com/douhashi/gh-project-promoter/issues/7",
									"body":       "Issue 2 body",
									"labels": map[string]interface{}{
										"nodes": []interface{}{
											map[string]interface{}{"name": "bug"},
											map[string]interface{}{"name": "urgent"},
										},
									},
								},
							},
							map[string]interface{}{
								"id": "PVTI_002",
								"fieldValues": map[string]interface{}{
									"nodes": []interface{}{
										map[string]interface{}{
											"__typename": "ProjectV2ItemFieldSingleSelectValue",
											"name":       "Backlog",
											"field": map[string]interface{}{
												"__typename": "ProjectV2SingleSelectField",
												"name":       "Status",
											},
										},
									},
								},
								"content": map[string]interface{}{
									"__typename": "Issue",
									"title":      "Sample Issue 1",
									"url":        "https://github.com/douhashi/gh-project-promoter/issues/6",
									"body":       "Issue 1 body",
									"labels": map[string]interface{}{
										"nodes": []interface{}{},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer srv.Close()

	client := newClientWithHTTP(srv.Client())
	client.inner = newTestGitHubV4Client(srv.URL, srv.Client())

	items, err := client.FetchProjectItems(context.Background(), "testuser", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	tests := []struct {
		idx    int
		id     string
		title  string
		url    string
		status string
		body   string
		labels []string
	}{
		{0, "PVTI_001", "Sample Issue 2", "https://github.com/douhashi/gh-project-promoter/issues/7", "Ready", "Issue 2 body", []string{"bug", "urgent"}},
		{1, "PVTI_002", "Sample Issue 1", "https://github.com/douhashi/gh-project-promoter/issues/6", "Backlog", "Issue 1 body", []string{}},
	}

	for _, tt := range tests {
		item := items[tt.idx]
		if item.ID != tt.id {
			t.Errorf("items[%d].ID = %q, want %q", tt.idx, item.ID, tt.id)
		}
		if item.Title != tt.title {
			t.Errorf("items[%d].Title = %q, want %q", tt.idx, item.Title, tt.title)
		}
		if item.URL != tt.url {
			t.Errorf("items[%d].URL = %q, want %q", tt.idx, item.URL, tt.url)
		}
		if item.Status != tt.status {
			t.Errorf("items[%d].Status = %q, want %q", tt.idx, item.Status, tt.status)
		}
		if item.Body != tt.body {
			t.Errorf("items[%d].Body = %q, want %q", tt.idx, item.Body, tt.body)
		}
		if len(item.Labels) != len(tt.labels) {
			t.Errorf("items[%d].Labels length = %d, want %d", tt.idx, len(item.Labels), len(tt.labels))
		} else {
			for i, l := range item.Labels {
				if l != tt.labels[i] {
					t.Errorf("items[%d].Labels[%d] = %q, want %q", tt.idx, i, l, tt.labels[i])
				}
			}
		}
	}
}

func TestFetchProjectItems_Pagination(t *testing.T) {
	var callCount atomic.Int32

	page1 := graphqlResponse{
		Data: map[string]interface{}{
			"user": map[string]interface{}{
				"projectV2": map[string]interface{}{
					"items": map[string]interface{}{
						"totalCount": 2,
						"pageInfo": map[string]interface{}{
							"hasNextPage": true,
							"endCursor":   "cursor1",
						},
						"nodes": []interface{}{
							map[string]interface{}{
								"id":          "PVTI_P1",
								"fieldValues": map[string]interface{}{"nodes": []interface{}{}},
								"content":     map[string]interface{}{"__typename": "Issue", "title": "Issue 1", "url": "https://example.com/1"},
							},
						},
					},
				},
			},
		},
	}

	page2 := graphqlResponse{
		Data: map[string]interface{}{
			"user": map[string]interface{}{
				"projectV2": map[string]interface{}{
					"items": map[string]interface{}{
						"totalCount": 2,
						"pageInfo": map[string]interface{}{
							"hasNextPage": false,
							"endCursor":   "",
						},
						"nodes": []interface{}{
							map[string]interface{}{
								"id":          "PVTI_P2",
								"fieldValues": map[string]interface{}{"nodes": []interface{}{}},
								"content":     map[string]interface{}{"__typename": "Issue", "title": "Issue 2", "url": "https://example.com/2"},
							},
						},
					},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		n := callCount.Add(1)
		var resp graphqlResponse
		if n == 1 {
			resp = page1
		} else {
			resp = page2
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer srv.Close()

	client := &Client{inner: newTestGitHubV4Client(srv.URL, srv.Client())}

	items, err := client.FetchProjectItems(context.Background(), "testuser", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	if items[0].ID != "PVTI_P1" {
		t.Errorf("items[0].ID = %q, want PVTI_P1", items[0].ID)
	}
	if items[1].ID != "PVTI_P2" {
		t.Errorf("items[1].ID = %q, want PVTI_P2", items[1].ID)
	}
}

func TestFetchProjectItems_OrgFallback(t *testing.T) {
	var callCount atomic.Int32

	userErrResp := graphqlResponse{
		Errors: []struct {
			Message string `json:"message"`
		}{{Message: "Could not resolve to a User"}},
	}

	orgResp := graphqlResponse{
		Data: map[string]interface{}{
			"organization": map[string]interface{}{
				"projectV2": map[string]interface{}{
					"items": map[string]interface{}{
						"totalCount": 1,
						"pageInfo": map[string]interface{}{
							"hasNextPage": false,
							"endCursor":   "",
						},
						"nodes": []interface{}{
							map[string]interface{}{
								"id":          "PVTI_ORG1",
								"fieldValues": map[string]interface{}{"nodes": []interface{}{}},
								"content":     map[string]interface{}{"__typename": "Issue", "title": "Org Issue", "url": "https://example.com/org/1"},
							},
						},
					},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		n := callCount.Add(1)
		if n == 1 {
			// First call: user query fails
			if err := json.NewEncoder(w).Encode(userErrResp); err != nil {
				t.Errorf("failed to encode response: %v", err)
			}
		} else {
			// Second call: org query succeeds
			if err := json.NewEncoder(w).Encode(orgResp); err != nil {
				t.Errorf("failed to encode response: %v", err)
			}
		}
	}))
	defer srv.Close()

	client := &Client{inner: newTestGitHubV4Client(srv.URL, srv.Client())}

	items, err := client.FetchProjectItems(context.Background(), "my-org", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	if items[0].Title != "Org Issue" {
		t.Errorf("items[0].Title = %q, want %q", items[0].Title, "Org Issue")
	}
}

func TestFetchProjectMeta_User(t *testing.T) {
	resp := graphqlResponse{
		Data: map[string]interface{}{
			"user": map[string]interface{}{
				"projectV2": map[string]interface{}{
					"id": "PVT_001",
					"field": map[string]interface{}{
						"__typename": "ProjectV2SingleSelectField",
						"id":         "PVTSSF_001",
						"options": []interface{}{
							map[string]interface{}{"id": "opt1", "name": "Backlog"},
							map[string]interface{}{"id": "opt2", "name": "Ready"},
							map[string]interface{}{"id": "opt3", "name": "In progress"},
							map[string]interface{}{"id": "opt4", "name": "Done"},
						},
					},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer srv.Close()

	client := &Client{inner: newTestGitHubV4Client(srv.URL, srv.Client())}

	meta, err := client.FetchProjectMeta(context.Background(), "testuser", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meta.ProjectID != "PVT_001" {
		t.Errorf("ProjectID = %q, want %q", meta.ProjectID, "PVT_001")
	}
	if meta.FieldID != "PVTSSF_001" {
		t.Errorf("FieldID = %q, want %q", meta.FieldID, "PVTSSF_001")
	}
	if len(meta.Options) != 4 {
		t.Fatalf("expected 4 options, got %d", len(meta.Options))
	}
	if meta.Options["Backlog"] != "opt1" {
		t.Errorf("Options[Backlog] = %q, want %q", meta.Options["Backlog"], "opt1")
	}
	if meta.Options["Ready"] != "opt2" {
		t.Errorf("Options[Ready] = %q, want %q", meta.Options["Ready"], "opt2")
	}
}

func TestFetchProjectMeta_OrgFallback(t *testing.T) {
	var callCount atomic.Int32

	userErrResp := graphqlResponse{
		Errors: []struct {
			Message string `json:"message"`
		}{{Message: "Could not resolve to a User"}},
	}

	orgResp := graphqlResponse{
		Data: map[string]interface{}{
			"organization": map[string]interface{}{
				"projectV2": map[string]interface{}{
					"id": "PVT_ORG1",
					"field": map[string]interface{}{
						"__typename": "ProjectV2SingleSelectField",
						"id":         "PVTSSF_ORG1",
						"options": []interface{}{
							map[string]interface{}{"id": "orgopt1", "name": "Todo"},
							map[string]interface{}{"id": "orgopt2", "name": "Done"},
						},
					},
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		n := callCount.Add(1)
		if n == 1 {
			if err := json.NewEncoder(w).Encode(userErrResp); err != nil {
				t.Errorf("failed to encode response: %v", err)
			}
		} else {
			if err := json.NewEncoder(w).Encode(orgResp); err != nil {
				t.Errorf("failed to encode response: %v", err)
			}
		}
	}))
	defer srv.Close()

	client := &Client{inner: newTestGitHubV4Client(srv.URL, srv.Client())}

	meta, err := client.FetchProjectMeta(context.Background(), "my-org", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if meta.ProjectID != "PVT_ORG1" {
		t.Errorf("ProjectID = %q, want %q", meta.ProjectID, "PVT_ORG1")
	}
	if meta.Options["Todo"] != "orgopt1" {
		t.Errorf("Options[Todo] = %q, want %q", meta.Options["Todo"], "orgopt1")
	}
}

func TestUpdateItemStatus(t *testing.T) {
	resp := graphqlResponse{
		Data: map[string]interface{}{
			"updateProjectV2ItemFieldValue": map[string]interface{}{
				"projectV2Item": map[string]interface{}{
					"id": "PVTI_001",
				},
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Errorf("failed to encode response: %v", err)
		}
	}))
	defer srv.Close()

	client := &Client{inner: newTestGitHubV4Client(srv.URL, srv.Client())}

	meta := &ProjectMeta{
		ProjectID: "PVT_001",
		FieldID:   "PVTSSF_001",
		Options:   map[string]string{"Ready": "opt2", "In progress": "opt3"},
	}

	err := client.UpdateItemStatus(context.Background(), meta, "PVTI_001", "Ready")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateItemStatus_InvalidStatus(t *testing.T) {
	client := &Client{inner: nil} // inner not needed since we error before API call

	meta := &ProjectMeta{
		ProjectID: "PVT_001",
		FieldID:   "PVTSSF_001",
		Options:   map[string]string{"Ready": "opt2"},
	}

	err := client.UpdateItemStatus(context.Background(), meta, "PVTI_001", "NonExistent")
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	want := `unknown status "NonExistent": not found in project options`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// newTestGitHubV4Client creates a githubv4.Client pointing at a test server.
func newTestGitHubV4Client(url string, httpClient *http.Client) *githubv4.Client {
	return githubv4.NewEnterpriseClient(url+"/graphql", httpClient)
}
