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
	}{
		{0, "PVTI_001", "Sample Issue 2", "https://github.com/douhashi/gh-project-promoter/issues/7", "Ready"},
		{1, "PVTI_002", "Sample Issue 1", "https://github.com/douhashi/gh-project-promoter/issues/6", "Backlog"},
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

// newTestGitHubV4Client creates a githubv4.Client pointing at a test server.
func newTestGitHubV4Client(url string, httpClient *http.Client) *githubv4.Client {
	return githubv4.NewEnterpriseClient(url+"/graphql", httpClient)
}
