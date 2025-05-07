package xkcd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"yadro.com/course/update/core"
)

type Client struct {
	log    *slog.Logger
	client http.Client
	url    string
}

var infoJSONndpoint = "/info.0.json"

func NewClient(url string, timeout time.Duration, log *slog.Logger) (*Client, error) {
	if url == "" {
		return nil, fmt.Errorf("empty base url specified")
	}
	return &Client{
		client: http.Client{Timeout: timeout},
		log:    log,
		url:    url,
	}, nil
}

func (c Client) Get(ctx context.Context, id int) (core.XKCDInfo, error) {

	requestUrl := c.url + "/" + strconv.Itoa(id) + infoJSONndpoint

	resp, err := c.client.Get(requestUrl)
	if err != nil {
		return core.XKCDInfo{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusNotFound {
			return core.XKCDInfo{}, fmt.Errorf("comic %d not found", id)
		}
		return core.XKCDInfo{}, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var result struct {
		Img        string `json:"img"`
		Title      string `json:"title"`
		Alt        string `json:"alt"`
		SafeTitle  string `json:"safe_title"`
		Transcript string `json:"transcript"`
	}

	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return core.XKCDInfo{}, fmt.Errorf("json decode failed: %w", err)
	}

	return core.XKCDInfo{ID: id,
		URL:         result.Img,
		Description: result.Alt + " " + result.Title + " " + result.SafeTitle + " " + result.Transcript}, nil

}

func (c Client) LastID(ctx context.Context) (int, error) {

	var id int
	requestUrl := c.url + infoJSONndpoint

	resp, err := c.client.Get(requestUrl)
	if err != nil {
		return id, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return id, err
	}

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return id, err
	}

	_, err = fmt.Sscan(fmt.Sprint(data["num"]), &id)

	return id, err
}
