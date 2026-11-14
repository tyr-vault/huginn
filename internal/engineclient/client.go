package engineclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rdvid/huginn/internal/models"
)

// Client talks to the internal scrape-engine.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func New(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type startResp struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"`
}

type jobResp struct {
	JobID       string `json:"job_id"`
	Status      string `json:"status"`
	Error       string `json:"error"`
	ResultCount int    `json:"result_count"`
}

// StartExtract kicks off an async extraction job.
func (c *Client) StartExtract(ctx context.Context, req models.SearchRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/extract", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	res, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("scrape-engine extract: %s", string(payload))
	}
	var out startResp
	if err := json.Unmarshal(payload, &out); err != nil {
		return "", err
	}
	return out.JobID, nil
}

// GetJob returns engine job status.
func (c *Client) GetJob(ctx context.Context, jobID string) (*jobResp, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/v1/jobs/"+jobID, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("scrape-engine job: %s", string(payload))
	}
	var out jobResp
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Results fetches completed businesses.
func (c *Client) Results(ctx context.Context, jobID string) ([]models.Business, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/v1/jobs/"+jobID+"/results", nil)
	if err != nil {
		return nil, err
	}
	// Results can be large / slow after scrape.
	client := &http.Client{Timeout: 2 * time.Minute}
	res, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("scrape-engine results: %s", string(payload))
	}
	var out []models.Business
	if err := json.Unmarshal(payload, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// WaitForCompletion polls until terminal status.
func (c *Client) WaitForCompletion(ctx context.Context, jobID string, every time.Duration) (*jobResp, error) {
	if every <= 0 {
		every = 3 * time.Second
	}
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		job, err := c.GetJob(ctx, jobID)
		if err != nil {
			return nil, err
		}
		switch job.Status {
		case models.JobCompleted, models.JobFailed:
			return job, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}
