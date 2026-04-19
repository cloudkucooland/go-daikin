package daikin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type AccessToken struct {
	Value     string    `json:"accessToken"`
	ExpiresIn int       `json:"accessTokenExpiresIn"`
	TokenType string    `json:"tokenType"`
	ExpiresAt time.Time // Calculated locally
}

type Client struct {
	Email        string
	DeveloperKey string // Developer API Key (from Developer Menu)
	APIKey       string // Integrator Token (from Home Integration menu)
	accessToken  AccessToken
	BaseURL      string // allow for alternate URLs for testing
	UserAgent    string
	Devices      []Device
	httpClient   *http.Client
	mu           sync.Mutex
}

type Location struct {
	LocationName string   `json:"locationName"`
	Devices      []Device `json:"devices"`
}

func New(ctx context.Context, email, developerKey, apiKey string) (*Client, error) {
	d := &Client{
		Email:        email,
		DeveloperKey: developerKey,
		APIKey:       apiKey,
		BaseURL:      base,
		UserAgent:    "cloudkucooland-go-daikin/1.0",
	}

	d.httpClient = &http.Client{Timeout: httpTimeout}

	// this gets the initial token
	d.mu.Lock()
	if err := d.refreshToken(ctx); err != nil {
		d.mu.Unlock()
		return nil, fmt.Errorf("failed to get initial token: %w", err)
	}
	d.mu.Unlock()

	if err := d.getDevices(ctx); err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}
	return d, nil
}

// getToken returns the current token, or fetches a new one if it's expired
func (d *Client) getToken(ctx context.Context) (string, error) {
	// Give ourselves a 60-second buffer so we don't expire mid-request
	d.mu.Lock()
	expired := time.Now().Add(60 * time.Second).After(d.accessToken.ExpiresAt)
	d.mu.Unlock()

	if expired {
		if err := d.refreshToken(ctx); err != nil {
			return "", fmt.Errorf("failed to refresh token: %w", err)
		}
	}

	d.mu.Lock()
	token := d.accessToken.Value
	d.mu.Unlock()
	return token, nil
}

func (d *Client) getDevices(ctx context.Context) error {
	res, err := d.doRequest(ctx, "GET", "/devices", nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("unexpected status: %s: %s", res.Status, body)
	}

	var locations []Location
	if err := json.NewDecoder(res.Body).Decode(&locations); err != nil {
		return fmt.Errorf("failed to decode locations: %w", err)
	}

	d.Devices = []Device{}
	for _, loc := range locations {
		for _, device := range loc.Devices {
			device.client = d
			d.Devices = append(d.Devices, device)
		}
	}
	return nil
}

func (d *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var lastErr error

	for i := 0; i < 3; i++ {
		res, err := d.doRequestOnce(ctx, method, path, body)
		if err != nil {
			lastErr = err
			continue
		}

		switch {
		case res.StatusCode == http.StatusTooManyRequests:
			// Retry after a brief pause
			bb, _ := io.ReadAll(res.Body)
			lastErr = fmt.Errorf("server error: %s %s", res.Status, bb)
			res.Body.Close()
			time.Sleep(2 * time.Second)
			continue
		case res.StatusCode >= 500:
			// Retry quickly on 5xx
			bb, _ := io.ReadAll(res.Body)
			lastErr = fmt.Errorf("server error: %s %s", res.Status, bb)
			res.Body.Close()
			time.Sleep(time.Duration(i+1) * 500 * time.Millisecond)
			continue
		case res.StatusCode >= 200 && res.StatusCode < 300:
			return res, nil
		default:
			// something else, call it quits
			return nil, fmt.Errorf("request failed %w", lastErr)
		}
	}

	return nil, fmt.Errorf("request failed after retries: %w", lastErr)
}

func (d *Client) doRequestOnce(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	token, err := d.getToken(ctx)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, err
		}
	}

	url := fmt.Sprintf("%s%s", d.BaseURL, path)
	req, err := http.NewRequestWithContext(ctx, method, url, &buf)
	if err != nil {
		return nil, err
	}

	req.Header["x-api-key"] = []string{d.APIKey} // requires explicit lowercase, do not use req.Header.Set()
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", d.UserAgent)

	return d.httpClient.Do(req)
}

func (d *Client) refreshToken(ctx context.Context) error {
	payload := struct {
		Email           string `json:"email"`
		IntegratorToken string `json:"integratorToken"`
	}{
		Email:           d.Email,
		IntegratorToken: d.DeveloperKey,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/token", d.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header["x-api-key"] = []string{d.APIKey} // requires explicit lowercase, do not use req.Header.Set()
	req.Header.Set("Content-Type", "application/json")

	res, err := d.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("auth failed: %s: %s", res.Status, body)
	}

	if err := json.NewDecoder(res.Body).Decode(&d.accessToken); err != nil {
		return err
	}

	d.accessToken.ExpiresAt = time.Now().Add(time.Duration(d.accessToken.ExpiresIn) * time.Second)
	return nil
}
