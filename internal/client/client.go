package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/k1LoW/gh-wait/internal/rule"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(addr string) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://%s", addr),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type StatusResponse struct {
	Version       string `json:"version"`
	PID           int    `json:"pid"`
	RuleCount     int    `json:"rule_count"`
	WatchingCount int    `json:"watching_count"`
}

func (c *Client) ProbeStatus() (*StatusResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/_/api/status")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var status StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}
	return &status, nil
}

func (c *Client) AddRule(r *rule.WatchRule) error {
	b, err := json.Marshal(r)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Post(c.baseURL+"/_/api/rules", "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to add rule: %s", string(body))
	}
	return nil
}

func (c *Client) ListRules() ([]*rule.WatchRule, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/_/api/rules")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var rules []*rule.WatchRule
	if err := json.NewDecoder(resp.Body).Decode(&rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (c *Client) DeleteRule(id string) error {
	req, err := http.NewRequest(http.MethodDelete, c.baseURL+"/_/api/rules/"+id, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete rule: %s", string(body))
	}
	return nil
}

func (c *Client) Shutdown() error {
	resp, err := c.httpClient.Post(c.baseURL+"/_/api/shutdown", "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
