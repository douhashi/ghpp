package github

import (
	"context"
	"fmt"

	"github.com/shurcooL/githubv4"
)

// ProjectInitializer abstracts the GraphQL operations needed to create a
// project from a template and link it to a repository.
type ProjectInitializer interface {
	GetProjectIDByOwnerAndNumber(ctx context.Context, owner string, number int) (string, error)
	GetOwnerID(ctx context.Context, owner string) (string, error)
	GetRepositoryID(ctx context.Context, owner, name string) (string, error)
	FindProjectByTitle(ctx context.Context, owner, title string) (*ExistingProject, error)
	CopyProjectV2(ctx context.Context, templateProjectID, ownerID, title string) (*NewProject, error)
	LinkProjectV2ToRepository(ctx context.Context, projectID, repositoryID string) error
	ListProjectV2Workflows(ctx context.Context, projectID string) ([]Workflow, error)
}

// --- GetProjectIDByOwnerAndNumber ---

type userProjectIDQuery struct {
	User struct {
		ProjectV2 struct {
			ID string
		} `graphql:"projectV2(number: $number)"`
	} `graphql:"user(login: $owner)"`
}

type orgProjectIDQuery struct {
	Organization struct {
		ProjectV2 struct {
			ID string
		} `graphql:"projectV2(number: $number)"`
	} `graphql:"organization(login: $owner)"`
}

// GetProjectIDByOwnerAndNumber resolves a ProjectV2 node ID from owner+number.
// Tries user first, then organization.
func (c *Client) GetProjectIDByOwnerAndNumber(ctx context.Context, owner string, number int) (string, error) {
	vars := map[string]interface{}{
		"owner":  githubv4.String(owner),
		"number": githubv4.Int(number),
	}
	var uq userProjectIDQuery
	userErr := c.inner.Query(ctx, &uq, vars)
	if userErr == nil && uq.User.ProjectV2.ID != "" {
		return uq.User.ProjectV2.ID, nil
	}
	var oq orgProjectIDQuery
	orgErr := c.inner.Query(ctx, &oq, vars)
	if orgErr == nil && oq.Organization.ProjectV2.ID != "" {
		return oq.Organization.ProjectV2.ID, nil
	}
	return "", fmt.Errorf("failed to resolve project ID for %s/%d (tried user and org): user: %v, org: %v", owner, number, userErr, orgErr)
}

// --- GetOwnerID ---

type userIDQuery struct {
	User struct {
		ID string
	} `graphql:"user(login: $login)"`
}

type orgIDQuery struct {
	Organization struct {
		ID string
	} `graphql:"organization(login: $login)"`
}

// GetOwnerID resolves the node ID of a user or organization.
// Tries user first, then organization.
func (c *Client) GetOwnerID(ctx context.Context, owner string) (string, error) {
	vars := map[string]interface{}{"login": githubv4.String(owner)}
	var uq userIDQuery
	userErr := c.inner.Query(ctx, &uq, vars)
	if userErr == nil && uq.User.ID != "" {
		return uq.User.ID, nil
	}
	var oq orgIDQuery
	orgErr := c.inner.Query(ctx, &oq, vars)
	if orgErr == nil && oq.Organization.ID != "" {
		return oq.Organization.ID, nil
	}
	return "", fmt.Errorf("failed to resolve owner ID for %q (tried user and org): user: %v, org: %v", owner, userErr, orgErr)
}

// --- GetRepositoryID ---

type repositoryIDQuery struct {
	Repository struct {
		ID string
	} `graphql:"repository(owner: $owner, name: $name)"`
}

// GetRepositoryID resolves the node ID of a repository.
func (c *Client) GetRepositoryID(ctx context.Context, owner, name string) (string, error) {
	var q repositoryIDQuery
	vars := map[string]interface{}{
		"owner": githubv4.String(owner),
		"name":  githubv4.String(name),
	}
	if err := c.inner.Query(ctx, &q, vars); err != nil {
		return "", fmt.Errorf("failed to resolve repository ID for %s/%s: %w", owner, name, err)
	}
	if q.Repository.ID == "" {
		return "", fmt.Errorf("repository %s/%s not found or not accessible", owner, name)
	}
	return q.Repository.ID, nil
}

// --- FindProjectByTitle ---

type projectListNode struct {
	ID     string
	Number int
	URL    string `graphql:"url"`
	Title  string
}

type userProjectListQuery struct {
	User struct {
		ProjectsV2 struct {
			PageInfo struct {
				HasNextPage bool
				EndCursor   githubv4.String
			}
			Nodes []projectListNode
		} `graphql:"projectsV2(first: 100, after: $cursor)"`
	} `graphql:"user(login: $owner)"`
}

type orgProjectListQuery struct {
	Organization struct {
		ProjectsV2 struct {
			PageInfo struct {
				HasNextPage bool
				EndCursor   githubv4.String
			}
			Nodes []projectListNode
		} `graphql:"projectsV2(first: 100, after: $cursor)"`
	} `graphql:"organization(login: $owner)"`
}

// FindProjectByTitle returns the first project owned by `owner` whose title
// exactly matches `title`. Returns (nil, nil) when no match exists.
// Tries user first, then organization.
func (c *Client) FindProjectByTitle(ctx context.Context, owner, title string) (*ExistingProject, error) {
	found, userErr := c.findUserProjectByTitle(ctx, owner, title)
	if userErr == nil {
		return found, nil
	}
	found, orgErr := c.findOrgProjectByTitle(ctx, owner, title)
	if orgErr != nil {
		return nil, fmt.Errorf("failed to list projects for %q (tried user and org): user: %v, org: %v", owner, userErr, orgErr)
	}
	return found, nil
}

func (c *Client) findUserProjectByTitle(ctx context.Context, owner, title string) (*ExistingProject, error) {
	var cursor *githubv4.String
	for {
		var q userProjectListQuery
		vars := map[string]interface{}{
			"owner":  githubv4.String(owner),
			"cursor": cursor,
		}
		if err := c.inner.Query(ctx, &q, vars); err != nil {
			return nil, err
		}
		for _, n := range q.User.ProjectsV2.Nodes {
			if n.Title == title {
				return &ExistingProject{ID: n.ID, Number: n.Number, URL: n.URL, Title: n.Title}, nil
			}
		}
		if !q.User.ProjectsV2.PageInfo.HasNextPage {
			return nil, nil
		}
		cursor = &q.User.ProjectsV2.PageInfo.EndCursor
	}
}

func (c *Client) findOrgProjectByTitle(ctx context.Context, owner, title string) (*ExistingProject, error) {
	var cursor *githubv4.String
	for {
		var q orgProjectListQuery
		vars := map[string]interface{}{
			"owner":  githubv4.String(owner),
			"cursor": cursor,
		}
		if err := c.inner.Query(ctx, &q, vars); err != nil {
			return nil, err
		}
		for _, n := range q.Organization.ProjectsV2.Nodes {
			if n.Title == title {
				return &ExistingProject{ID: n.ID, Number: n.Number, URL: n.URL, Title: n.Title}, nil
			}
		}
		if !q.Organization.ProjectsV2.PageInfo.HasNextPage {
			return nil, nil
		}
		cursor = &q.Organization.ProjectsV2.PageInfo.EndCursor
	}
}

// --- CopyProjectV2 ---

type copyProjectV2Mutation struct {
	CopyProjectV2 struct {
		ProjectV2 struct {
			ID     string
			Number int
			URL    string `graphql:"url"`
			Title  string
		}
	} `graphql:"copyProjectV2(input: $input)"`
}

// CopyProjectV2 copies a template project to the given owner with a new title.
func (c *Client) CopyProjectV2(ctx context.Context, templateProjectID, ownerID, title string) (*NewProject, error) {
	var m copyProjectV2Mutation
	include := githubv4.Boolean(false)
	input := githubv4.CopyProjectV2Input{
		ProjectID:          githubv4.ID(templateProjectID),
		OwnerID:            githubv4.ID(ownerID),
		Title:              githubv4.String(title),
		IncludeDraftIssues: &include,
	}
	if err := c.inner.Mutate(ctx, &m, input, nil); err != nil {
		return nil, fmt.Errorf("failed to copy project: %w", err)
	}
	return &NewProject{
		ID:     m.CopyProjectV2.ProjectV2.ID,
		Number: m.CopyProjectV2.ProjectV2.Number,
		URL:    m.CopyProjectV2.ProjectV2.URL,
		Title:  m.CopyProjectV2.ProjectV2.Title,
	}, nil
}

// --- LinkProjectV2ToRepository ---

type linkProjectV2ToRepositoryMutation struct {
	LinkProjectV2ToRepository struct {
		Repository struct {
			ID string
		}
	} `graphql:"linkProjectV2ToRepository(input: $input)"`
}

// LinkProjectV2ToRepository links a ProjectV2 to a repository.
func (c *Client) LinkProjectV2ToRepository(ctx context.Context, projectID, repositoryID string) error {
	var m linkProjectV2ToRepositoryMutation
	input := githubv4.LinkProjectV2ToRepositoryInput{
		ProjectID:    githubv4.ID(projectID),
		RepositoryID: githubv4.ID(repositoryID),
	}
	if err := c.inner.Mutate(ctx, &m, input, nil); err != nil {
		return fmt.Errorf("failed to link project to repository: %w", err)
	}
	return nil
}

// --- ListProjectV2Workflows ---

type projectV2WorkflowsQuery struct {
	Node struct {
		ProjectV2 struct {
			Workflows struct {
				Nodes []struct {
					Name    string
					Number  int
					Enabled bool
				}
			} `graphql:"workflows(first: 50)"`
		} `graphql:"... on ProjectV2"`
	} `graphql:"node(id: $id)"`
}

// ListProjectV2Workflows returns the workflows attached to a ProjectV2.
func (c *Client) ListProjectV2Workflows(ctx context.Context, projectID string) ([]Workflow, error) {
	var q projectV2WorkflowsQuery
	vars := map[string]interface{}{"id": githubv4.ID(projectID)}
	if err := c.inner.Query(ctx, &q, vars); err != nil {
		return nil, fmt.Errorf("failed to list project workflows: %w", err)
	}
	out := make([]Workflow, 0, len(q.Node.ProjectV2.Workflows.Nodes))
	for _, n := range q.Node.ProjectV2.Workflows.Nodes {
		out = append(out, Workflow{Name: n.Name, Number: n.Number, Enabled: n.Enabled})
	}
	return out, nil
}

// Compile-time check that Client implements ProjectInitializer.
var _ ProjectInitializer = (*Client)(nil)
