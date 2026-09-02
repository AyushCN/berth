package ml

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/AyushCN/berth/internal/domain"
)

// Client implements domain.PredictionService via HTTP.
type Client struct {
	baseURL string
	client  *http.Client
}

// NewClient creates a prediction client.
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://localhost:50052"
	}
	return &Client{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Predict analyzes a repository and returns a runtime profile.
func (c *Client) Predict(ctx context.Context, gitURL string, branch string, localPath string) (*domain.RuntimeProfile, error) {
	reqBody, _ := json.Marshal(map[string]string{
		"git_url":    gitURL,
		"branch":     branch,
		"local_path": localPath,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/predict", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prediction service unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("prediction service returned %d", resp.StatusCode)
	}

	var result struct {
		Language    string  `json:"language"`
		BaseImage   string  `json:"base_image"`
		InstallCmd  string  `json:"install_cmd"`
		StartCmd    string  `json:"start_cmd"`
		Port        int     `json:"port"`
		NeedsDB     bool    `json:"needs_db"`
		Confidence  float64 `json:"confidence"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode prediction response: %w", err)
	}

	return &domain.RuntimeProfile{
		Language:    result.Language,
		BaseImage:   result.BaseImage,
		InstallCmd:  result.InstallCmd,
		StartCmd:    result.StartCmd,
		ExposedPort: result.Port,
		NeedsDB:     result.NeedsDB,
		Confidence:  result.Confidence,
	}, nil
}
