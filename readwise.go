package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type ReadwiseClient struct {
	client    *http.Client
	apiKey    string
	baseURL   string
	rateLimit *time.Ticker // For rate limiting
}

type ReadwiseHighlight struct {
	Text          string `json:"text"`
	Title         string `json:"title"`
	Author        string `json:"author"`
	SourceType    string `json:"source_type"`
	Category      string `json:"category"`
	Note          string `json:"note,omitempty"`
	HighlightedAt string `json:"highlighted_at,omitempty"`
	Location      int    `json:"location,omitempty"`
	LocationType  string `json:"location_type,omitempty"`
	SourceURL     string `json:"source_url,omitempty"`
}

type ReadwiseRequest struct {
	Highlights []ReadwiseHighlight `json:"highlights"`
}

func NewReadwiseClient(apiKey string) *ReadwiseClient {
	return &ReadwiseClient{
		client:    &http.Client{},
		apiKey:    apiKey,
		baseURL:   "https://readwise.io/api/v2/highlights/",
		rateLimit: time.NewTicker(time.Minute / 240), // 240 requests per minute
	}
}

func (c *ReadwiseClient) SendHighlights(highlights []ReadwiseHighlight) error {
	// Split highlights into batches of 100 to avoid large payloads
	const batchSize = 100
	for i := 0; i < len(highlights); i += batchSize {
		end := i + batchSize
		if end > len(highlights) {
			end = len(highlights)
		}

		batch := highlights[i:end]
		if err := c.sendBatch(batch); err != nil {
			return fmt.Errorf("error sending batch %d-%d: %w", i, end, err)
		}
	}
	return nil
}

func (c *ReadwiseClient) sendBatch(highlights []ReadwiseHighlight) error {
	// Wait for rate limit ticker
	<-c.rateLimit.C

	payload := ReadwiseRequest{
		Highlights: highlights,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling highlights: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Token %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	maxRetries := 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := c.client.Do(req)
		if err != nil {
			return fmt.Errorf("error making request: %w", err)
		}
		defer resp.Body.Close()

		// Handle rate limiting
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := resp.Header.Get("Retry-After")
			seconds, err := strconv.Atoi(retryAfter)
			if err != nil {
				seconds = 60 // Default to 60 seconds if header is missing or invalid
			}

			if attempt < maxRetries {
				fmt.Fprintf(os.Stderr, "Rate limited, waiting %d seconds...\n", seconds)
				time.Sleep(time.Duration(seconds) * time.Second)
				continue
			}
			return fmt.Errorf("max retries exceeded after rate limiting")
		}

		// Handle other errors
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
		}

		return nil
	}

	return fmt.Errorf("max retries exceeded")
}
