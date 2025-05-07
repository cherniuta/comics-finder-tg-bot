package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"yadro.com/course/telegram/core"
)

type APIClient struct {
	baseURL    string
	httpClient *http.Client
	log        *slog.Logger
}

func New(baseURL string, logger *slog.Logger) *APIClient {
	return &APIClient{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 90 * time.Second},
		log:        logger,
	}
}

func (c *APIClient) Search(ctx context.Context, limit int, words string) (core.SearchResult, error) {
	params := url.Values{}
	params.Add("limit", strconv.Itoa(limit))
	params.Add("phrase", words)

	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		c.baseURL+"/api/search?"+params.Encode(),
		nil,
	)
	if err != nil {
		return core.SearchResult{}, fmt.Errorf("create request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return core.SearchResult{}, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return core.SearchResult{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result core.SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return core.SearchResult{}, fmt.Errorf("decode response failed: %w", err)
	}

	return result, nil
}

func (c *APIClient) Update(ctx context.Context) (err error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/db/update", nil)
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

type LoginBody struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func (c *APIClient) Login(ctx context.Context, user, password string) (string, error) {
	loginBody := LoginBody{Name: user, Password: password}

	jsonData, err := json.Marshal(loginBody)
	if err != nil {
		c.log.Error("problem of decoding", "error", err)
		return "", err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/api/login",
		bytes.NewBuffer(jsonData),
	)

	if err != nil {
		c.log.Error("create request failed", "error", err)
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Error("API request failed", "error", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code")
	}

	tokenBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.log.Error("read body failed", "error", err)
		return "", fmt.Errorf("read body failed: %w", err)
	}

	token := string(tokenBytes)

	return token, nil

}

func (c *APIClient) UpdateComics(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseURL+"/api/db/update",
		nil,
	)
	if err != nil {
		c.log.Error("create update request failed", "error", err)
		return fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Authorization", "Token "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Error("update request failed", "error", err)
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		c.log.Info("database update successful")
		return nil

	case http.StatusAccepted:
		body, _ := io.ReadAll(resp.Body)
		c.log.Warn("database update already in progress",
			slog.String("response", string(body)),
		)
		return core.ErrAlreadyExists

	case http.StatusUnauthorized:
		c.log.Error("update failed - unauthorized",
			slog.Int("status_code", resp.StatusCode),
		)
		return core.ErrUnauthorized

	default:
		body, _ := io.ReadAll(resp.Body)
		c.log.Error("unexpected status code from update endpoint",
			slog.Int("status_code", resp.StatusCode),
			slog.String("response", string(body)),
		)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

func (c *APIClient) Drop(ctx context.Context, token string) error {
	req, err := http.NewRequestWithContext(
		ctx,
		"DELETE",
		c.baseURL+"/api/db",
		nil,
	)
	if err != nil {
		c.log.Error("create drop request failed", "error", err)
		return fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Authorization", "Token "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Error("drop request failed", "error", err)
		return fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		c.log.Info("database was successfully deleted")
		return nil

	case http.StatusUnauthorized:
		c.log.Error("drop failed - unauthorized",
			slog.Int("status_code", resp.StatusCode),
		)
		return core.ErrUnauthorized

	default:
		body, _ := io.ReadAll(resp.Body)
		c.log.Error("unexpected status code from drop endpoint",
			slog.Int("status_code", resp.StatusCode),
			slog.String("response", string(body)),
		)
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

type Stats struct {
	WordsTotal    int `json:"words_total"`
	WordsUnique   int `json:"words_unique"`
	ComicsFetched int `json:"comics_fetched"`
	ComicsTotal   int `json:"comics_total"`
}

func (c *APIClient) Stats(ctx context.Context, token string) (core.StatsResult, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		c.baseURL+"/api/db/stats",
		nil,
	)
	if err != nil {
		c.log.Error("create stats request failed", "error", err)
		return core.StatsResult{}, fmt.Errorf("create request failed: %w", err)
	}

	req.Header.Set("Authorization", "Token "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.log.Error("stats request failed", "error", err)
		return core.StatsResult{}, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		c.log.Info("database was successfully deleted")

		var result Stats
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return core.StatsResult{}, fmt.Errorf("decode response failed: %w", err)
		}
		return core.StatsResult{
			WordsTotal:    result.WordsTotal,
			WordsUnique:   result.WordsUnique,
			ComicsFetched: result.ComicsFetched,
			ComicsTotal:   result.ComicsTotal}, nil

	case http.StatusUnauthorized:
		c.log.Error("stats failed - unauthorized",
			slog.Int("status_code", resp.StatusCode),
		)
		return core.StatsResult{}, core.ErrUnauthorized

	default:
		body, _ := io.ReadAll(resp.Body)
		c.log.Error("unexpected status code from stats endpoint",
			slog.Int("status_code", resp.StatusCode),
			slog.String("response", string(body)),
		)
		return core.StatsResult{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}
