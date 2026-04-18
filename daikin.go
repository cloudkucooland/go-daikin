package daikin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	// "log/slog"
	"net/http"
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
	AccessToken  AccessToken
	Devices      []Device
}

type Location struct {
	LocationName string   `json:"locationName"`
	Devices      []Device `json:"devices"`
}

func New(email, developerKey, apiKey string) (*Client, error) {
	d := &Client{
		Email:        email,
		DeveloperKey: developerKey,
		APIKey:       apiKey,
	}

	ctx := context.Background()

	d.refreshToken(ctx)
	// slog.Info("Daikin authenticated", "expires_at", d.AccessToken.ExpiresAt)

	if err := d.getDevices(ctx); err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}
	return d, nil
}

// GetToken returns the current token, or fetches a new one if it's expired
func (d *Client) GetToken(ctx context.Context) (string, error) {
	// Give ourselves a 60-second buffer so we don't expire mid-request
	if time.Now().Add(60 * time.Second).Before(d.AccessToken.ExpiresAt) {
		return d.AccessToken.Value, nil
	}
	d.refreshToken(ctx)
	// slog.Info("Daikin token renewed", "expires_at", d.AccessToken.ExpiresAt)
	return d.AccessToken.Value, nil
}

func (d *Client) getDevices(ctx context.Context) error {
	res, err := d.doRequest(ctx, "GET", "/devices", nil)
	if err != nil {
		return err
	}
	defer res.Body.Close()

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

	/* for _, dd := range d.Devices {
		slog.Info("Daikin device found", "name", dd.Name, "id", dd.ID, "model", dd.Model)
	} */
	return nil
}

func (d *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	token, err := d.GetToken(ctx)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, err
		}
	}

	url := fmt.Sprintf("%s%s", base, path)
	req, err := http.NewRequestWithContext(ctx, method, url, &buf)
	if err != nil {
		return nil, err
	}

	req.Header["x-api-key"] = []string{d.APIKey}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	// req.Header.Set("User-Agent", "DeepCool/1.0")

	client := &http.Client{Timeout: httpTimeout}
	return client.Do(req)
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
	url := fmt.Sprintf("%s/token", base)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header["x-api-key"] = []string{d.APIKey}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: httpTimeout}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("auth failed: %s", res.Status)
	}

	if err := json.NewDecoder(res.Body).Decode(&d.AccessToken); err != nil {
		return err
	}

	d.AccessToken.ExpiresAt = time.Now().Add(time.Duration(d.AccessToken.ExpiresIn) * time.Second)
	return nil
}
