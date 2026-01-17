package clinvar

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mkoziy/genome/exporter/internal/ratelimit"
)

const defaultBaseURL = "https://eutils.ncbi.nlm.nih.gov/entrez/eutils"

var baseURL = defaultBaseURL

const (
	toolName = "snp-downloader"
)

// Client handles ClinVar API requests.
type Client struct {
	httpClient *http.Client
	limiter    ratelimit.Limiter
	apiKey     string
	email      string
}

// NewClient creates a new ClinVar client.
func NewClient(limiter ratelimit.Limiter, apiKey, email string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		limiter:    limiter,
		apiKey:     apiKey,
		email:      email,
	}
}

// Search performs an ESearch query.
func (c *Client) Search(ctx context.Context, query string, retStart, retMax int) (*SearchResponse, error) {
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("db", "clinvar")
	params.Set("term", query)
	params.Set("retstart", fmt.Sprintf("%d", retStart))
	params.Set("retmax", fmt.Sprintf("%d", retMax))
	params.Set("retmode", "json")
	params.Set("tool", toolName)
	if c.email != "" {
		params.Set("email", c.email)
	}
	if c.apiKey != "" {
		params.Set("api_key", c.apiKey)
	}

	u := fmt.Sprintf("%s/esearch.fcgi?%s", baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result struct {
		ESearchResult SearchResponse `json:"esearchresult"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result.ESearchResult, nil
}

// Fetch retrieves full variant details by IDs.
func (c *Client) Fetch(ctx context.Context, ids []string) ([]ClinVarSet, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	if err := c.limiter.Wait(ctx); err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("db", "clinvar")
	params.Set("id", joinIDs(ids))
	params.Set("rettype", "vcv")
	params.Set("retmode", "xml")
	params.Set("tool", toolName)
	if c.email != "" {
		params.Set("email", c.email)
	}
	if c.apiKey != "" {
		params.Set("api_key", c.apiKey)
	}

	u := fmt.Sprintf("%s/efetch.fcgi?%s", baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var wrapper struct {
		XMLName xml.Name     `xml:"ClinVarResult-Set"`
		Sets    []ClinVarSet `xml:"ClinVarSet"`
	}
	if err := xml.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("decode XML: %w", err)
	}
	return wrapper.Sets, nil
}

func joinIDs(ids []string) string {
	return strings.Join(ids, ",")
}
