package gitlab

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/loan-mgt/dumb-flyway-validate/internal/config"
	"github.com/loan-mgt/dumb-flyway-validate/pkg/errors"
)

// Client represents a GitLab API client
type Client struct {
	config *config.GitLabConfig
	client *http.Client
	debug  bool
}

// MergeRequestInfo represents GitLab merge request information
type MergeRequestInfo struct {
	DiffRefs struct {
		BaseSHA  string `json:"base_sha"`
		HeadSHA  string `json:"head_sha"`
		StartSHA string `json:"start_sha"`
	} `json:"diff_refs"`
}

// NewClient creates a new GitLab client
func NewClient(cfg *config.GitLabConfig) *Client {
	return &Client{
		config: cfg,
		client: &http.Client{},
	}
}

// GetMergeRequestInfo retrieves merge request information from GitLab
func (c *Client) GetMergeRequestInfo() (*MergeRequestInfo, error) {
	url := fmt.Sprintf("%s/v4/projects/%s/merge_requests/%s",
		c.config.URL, c.config.ProjectID, c.config.MRIID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", c.config.Token)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitLab API returned status %d", resp.StatusCode)
	}

	var mrInfo MergeRequestInfo
	if err := json.NewDecoder(resp.Body).Decode(&mrInfo); err != nil {
		return nil, fmt.Errorf("failed to decode merge request info: %w", err)
	}

	return &mrInfo, nil
}

// PostDiscussion posts a discussion comment to the merge request
func (c *Client) PostDiscussion(validationError errors.ValidationError, mrInfo *MergeRequestInfo) error {
	url := fmt.Sprintf("%s/v4/projects/%s/merge_requests/%s/discussions",
		c.config.URL, c.config.ProjectID, c.config.MRIID)

	payload := map[string]interface{}{
		"body": validationError.MarkdownMessage(),
		"position": map[string]interface{}{
			"position_type":            "file",
			"base_sha":                 mrInfo.DiffRefs.BaseSHA,
			"head_sha":                 mrInfo.DiffRefs.HeadSHA,
			"start_sha":                mrInfo.DiffRefs.StartSHA,
			"new_path":                 validationError.NewPath,
			"old_path":                 validationError.OldPath,
			"old_line":                 nil,
			"new_line":                 nil,
			"line_range":               map[string]interface{}{},
			"ignore_whitespace_change": false,
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", c.config.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("GitLab API returned status %d when posting discussion", resp.StatusCode)
	}

	return nil
}

// PostValidationErrors posts all validation errors to GitLab
func (c *Client) PostValidationErrors(validationErrors []errors.ValidationError) error {
	mrInfo, err := c.GetMergeRequestInfo()
	if err != nil {
		return fmt.Errorf("failed to get merge request info: %w", err)
	}

	for _, validationError := range validationErrors {
		if err := c.PostDiscussion(validationError, mrInfo); err != nil {
			return fmt.Errorf("failed to post discussion for %s: %w", validationError.NewPath, err)
		}
	}

	return nil
}
