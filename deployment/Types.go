package deployment

import (
	"fmt"
	"time"
)

type Repo struct {
	Owner       string `json:"owner,omitempty"`
	Name        string `json:"name,omitempty"`
	Environment string `json:"environment,omitempty"`
}

type Deployment struct {
	// deployment
	ID            *int64     `json:"id,omitempty"`
	DeploymentUrl *string    `json:"deployment_url,omitempty"`
	SHA           *string    `json:"sha,omitempty"`
	CreatedAt     *time.Time `json:"created_at,omitempty"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty"`

	// commits
	ComparisonUrl  *string   `json:"comparison_url,omitempty"`
	CommitsAdded   []*Commit `json:"commits_added,omitempty"`
	CommitsRemoved []*Commit `json:"commits_removed,omitempty"`
}

// GetID returns the ID field if it's non-nil, zero value otherwise.
// copied from github.Deployment.GetID
func (d *Deployment) GetID() int64 {
	if d == nil || d.ID == nil {
		return 0
	}
	return *d.ID
}

// GetSHA returns the SHA field if it's non-nil, zero value otherwise.
func (d *Deployment) GetSHA() string {
	if d == nil || d.SHA == nil {
		return ""
	}
	return *d.SHA
}

// GetCreatedAt returns the CreatedAt field if it's non-nil, zero value otherwise.
func (d *Deployment) GetCreatedAt() time.Time {
	if d == nil || d.CreatedAt == nil {
		return time.Time{}
	}
	return *d.CreatedAt
}

func (d *Deployment) String() string {
	if d == nil {
		return "<nil>"
	}

	return fmt.Sprintf(
		"Deployment(id: %d, sha: %q, created: %v, commits: +%v/-%v)",
		d.GetID(),
		d.GetSHA(),
		d.GetCreatedAt(),
		d.CommitsAdded,
		d.CommitsRemoved,
	)
}

type Commit struct {
	SHA   string `json:"sha,omitempty"`
	Title string `json:"title,omitempty"`
	URL   string `json:"url,omitempty"`
}

func (c *Commit) String() string {
	if c == nil {
		return "<nil>"
	}
	return fmt.Sprintf(
		"Commit(sha: %q, title: %s, url: %s",
		c.SHA,
		c.Title,
		c.URL,
	)
}
