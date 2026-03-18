package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

// ItemFetcher abstracts fetching project items for testability.
type ItemFetcher interface {
	FetchProjectItems(ctx context.Context, owner string, projectNumber int) ([]ProjectItem, error)
}

// ItemPromoter abstracts promoting project items for testability.
type ItemPromoter interface {
	FetchProjectMeta(ctx context.Context, owner string, projectNumber int) (*ProjectMeta, error)
	UpdateItemStatus(ctx context.Context, meta *ProjectMeta, itemID string, statusName string) error
}

// Client wraps the GitHub GraphQL API client.
type Client struct {
	inner *githubv4.Client
}

// NewClient creates a Client authenticated with the given token.
func NewClient(token string) *Client {
	src := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), src)
	return &Client{inner: githubv4.NewClient(httpClient)}
}

// newClientWithHTTP creates a Client backed by the given http.Client (for testing).
func newClientWithHTTP(httpClient *http.Client) *Client {
	return &Client{inner: githubv4.NewClient(httpClient)}
}

// projectV2Query is the GraphQL query struct for fetching project items.
// It works for both user and organization owners via the ownerType parameter.
type projectV2Query struct {
	User struct {
		ProjectV2 struct {
			Items struct {
				TotalCount int
				PageInfo   struct {
					HasNextPage bool
					EndCursor   githubv4.String
				}
				Nodes []itemNode
			} `graphql:"items(first: 100, after: $cursor)"`
		} `graphql:"projectV2(number: $number)"`
	} `graphql:"user(login: $owner)"`
}

type orgProjectV2Query struct {
	Organization struct {
		ProjectV2 struct {
			Items struct {
				TotalCount int
				PageInfo   struct {
					HasNextPage bool
					EndCursor   githubv4.String
				}
				Nodes []itemNode
			} `graphql:"items(first: 100, after: $cursor)"`
		} `graphql:"projectV2(number: $number)"`
	} `graphql:"organization(login: $owner)"`
}

type itemNode struct {
	ID          string
	FieldValues struct {
		Nodes []fieldValueNode
	} `graphql:"fieldValues(first: 20)"`
	Content itemContent `graphql:"content"`
}

type fieldValueNode struct {
	TypeName           string `graphql:"__typename"`
	ProjectV2ItemField struct {
		Name  string
		Field struct {
			TypeName              string `graphql:"__typename"`
			ProjectV2SingleSelect struct {
				Name string
			} `graphql:"... on ProjectV2SingleSelectField"`
		}
	} `graphql:"... on ProjectV2ItemFieldSingleSelectValue"`
}

type itemContent struct {
	TypeName string `graphql:"__typename"`
	Issue    struct {
		Title string
		URL   string `graphql:"url"`
	} `graphql:"... on Issue"`
	PullRequest struct {
		Title string
		URL   string `graphql:"url"`
	} `graphql:"... on PullRequest"`
}

// FetchProjectItems retrieves all items from a GitHub ProjectV2.
// It first tries as a user project; if that fails, it retries as an organization project.
func (c *Client) FetchProjectItems(ctx context.Context, owner string, projectNumber int) ([]ProjectItem, error) {
	items, err := c.fetchUserProjectItems(ctx, owner, projectNumber)
	if err == nil {
		return items, nil
	}

	orgItems, orgErr := c.fetchOrgProjectItems(ctx, owner, projectNumber)
	if orgErr != nil {
		return nil, fmt.Errorf("failed to fetch project items (tried user and org): user: %w, org: %v", err, orgErr)
	}
	return orgItems, nil
}

func (c *Client) fetchUserProjectItems(ctx context.Context, owner string, projectNumber int) ([]ProjectItem, error) {
	var allItems []ProjectItem
	var cursor *githubv4.String

	for {
		var q projectV2Query
		variables := map[string]interface{}{
			"owner":  githubv4.String(owner),
			"number": githubv4.Int(projectNumber),
			"cursor": cursor,
		}

		if err := c.inner.Query(ctx, &q, variables); err != nil {
			return nil, fmt.Errorf("failed to query user project: %w", err)
		}

		for _, node := range q.User.ProjectV2.Items.Nodes {
			allItems = append(allItems, toProjectItem(node))
		}

		if !q.User.ProjectV2.Items.PageInfo.HasNextPage {
			break
		}
		cursor = &q.User.ProjectV2.Items.PageInfo.EndCursor
	}

	return allItems, nil
}

func (c *Client) fetchOrgProjectItems(ctx context.Context, owner string, projectNumber int) ([]ProjectItem, error) {
	var allItems []ProjectItem
	var cursor *githubv4.String

	for {
		var q orgProjectV2Query
		variables := map[string]interface{}{
			"owner":  githubv4.String(owner),
			"number": githubv4.Int(projectNumber),
			"cursor": cursor,
		}

		if err := c.inner.Query(ctx, &q, variables); err != nil {
			return nil, fmt.Errorf("failed to query org project: %w", err)
		}

		for _, node := range q.Organization.ProjectV2.Items.Nodes {
			allItems = append(allItems, toProjectItem(node))
		}

		if !q.Organization.ProjectV2.Items.PageInfo.HasNextPage {
			break
		}
		cursor = &q.Organization.ProjectV2.Items.PageInfo.EndCursor
	}

	return allItems, nil
}

// statusFieldNode represents the Status field inline fragment in project metadata queries.
type statusFieldNode struct {
	TypeName                   string `graphql:"__typename"`
	ProjectV2SingleSelectField struct {
		ID      string
		Options []struct {
			ID   string
			Name string
		}
	} `graphql:"... on ProjectV2SingleSelectField"`
}

// projectMetaQuery is the GraphQL query struct for fetching project metadata (user).
type projectMetaQuery struct {
	User struct {
		ProjectV2 struct {
			ID    string
			Field statusFieldNode `graphql:"field(name: \"Status\")"`
		} `graphql:"projectV2(number: $number)"`
	} `graphql:"user(login: $owner)"`
}

// orgProjectMetaQuery is the GraphQL query struct for fetching project metadata (org).
type orgProjectMetaQuery struct {
	Organization struct {
		ProjectV2 struct {
			ID    string
			Field statusFieldNode `graphql:"field(name: \"Status\")"`
		} `graphql:"projectV2(number: $number)"`
	} `graphql:"organization(login: $owner)"`
}

// updateItemStatusMutation is the GraphQL mutation struct for updating an item's status.
type updateItemStatusMutation struct {
	UpdateProjectV2ItemFieldValue struct {
		ProjectV2Item struct {
			ID string
		}
	} `graphql:"updateProjectV2ItemFieldValue(input: $input)"`
}

// FetchProjectMeta retrieves project-level metadata (ID, Status field ID, and options).
// It first tries as a user project; if that fails, it retries as an organization project.
func (c *Client) FetchProjectMeta(ctx context.Context, owner string, projectNumber int) (*ProjectMeta, error) {
	meta, err := c.fetchUserProjectMeta(ctx, owner, projectNumber)
	if err == nil {
		return meta, nil
	}

	orgMeta, orgErr := c.fetchOrgProjectMeta(ctx, owner, projectNumber)
	if orgErr != nil {
		return nil, fmt.Errorf("failed to fetch project meta (tried user and org): user: %w, org: %v", err, orgErr)
	}
	return orgMeta, nil
}

func (c *Client) fetchUserProjectMeta(ctx context.Context, owner string, projectNumber int) (*ProjectMeta, error) {
	var q projectMetaQuery
	variables := map[string]interface{}{
		"owner":  githubv4.String(owner),
		"number": githubv4.Int(projectNumber),
	}

	if err := c.inner.Query(ctx, &q, variables); err != nil {
		return nil, fmt.Errorf("failed to query user project meta: %w", err)
	}

	return toProjectMeta(q.User.ProjectV2.ID, q.User.ProjectV2.Field), nil
}

func (c *Client) fetchOrgProjectMeta(ctx context.Context, owner string, projectNumber int) (*ProjectMeta, error) {
	var q orgProjectMetaQuery
	variables := map[string]interface{}{
		"owner":  githubv4.String(owner),
		"number": githubv4.Int(projectNumber),
	}

	if err := c.inner.Query(ctx, &q, variables); err != nil {
		return nil, fmt.Errorf("failed to query org project meta: %w", err)
	}

	return toProjectMeta(q.Organization.ProjectV2.ID, q.Organization.ProjectV2.Field), nil
}

func toProjectMeta(projectID string, field statusFieldNode) *ProjectMeta {
	ssf := field.ProjectV2SingleSelectField
	options := make(map[string]string, len(ssf.Options))
	for _, opt := range ssf.Options {
		options[opt.Name] = opt.ID
	}
	return &ProjectMeta{
		ProjectID: projectID,
		FieldID:   ssf.ID,
		Options:   options,
	}
}

// UpdateItemStatus updates the Status field of a project item to the given status name.
func (c *Client) UpdateItemStatus(ctx context.Context, meta *ProjectMeta, itemID string, statusName string) error {
	optionID, ok := meta.Options[statusName]
	if !ok {
		return fmt.Errorf("unknown status %q: not found in project options", statusName)
	}

	var m updateItemStatusMutation
	input := githubv4.UpdateProjectV2ItemFieldValueInput{
		ProjectID: githubv4.ID(meta.ProjectID),
		ItemID:    githubv4.ID(itemID),
		FieldID:   githubv4.ID(meta.FieldID),
		Value: githubv4.ProjectV2FieldValue{
			SingleSelectOptionID: githubv4.NewString(githubv4.String(optionID)),
		},
	}

	if err := c.inner.Mutate(ctx, &m, input, nil); err != nil {
		return fmt.Errorf("failed to update item status to %q: %w", statusName, err)
	}

	return nil
}

func toProjectItem(node itemNode) ProjectItem {
	item := ProjectItem{ID: node.ID}

	switch node.Content.TypeName {
	case "Issue":
		item.Title = node.Content.Issue.Title
		item.URL = node.Content.Issue.URL
	case "PullRequest":
		item.Title = node.Content.PullRequest.Title
		item.URL = node.Content.PullRequest.URL
	}

	for _, fv := range node.FieldValues.Nodes {
		if fv.TypeName == "ProjectV2ItemFieldSingleSelectValue" &&
			fv.ProjectV2ItemField.Field.ProjectV2SingleSelect.Name == "Status" {
			item.Status = fv.ProjectV2ItemField.Name
			break
		}
	}

	return item
}
