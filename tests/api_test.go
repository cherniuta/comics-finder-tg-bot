package words_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const address = "http://localhost:28080"

var client = http.Client{
	Timeout: 5 * time.Minute,
}

func TestPreflight(t *testing.T) {
	require.Equal(t, true, true)
}

type PingResponse struct {
	Replies map[string]string `json:"replies"`
}

func TestPing(t *testing.T) {
	resp, err := client.Get(address + "/api/ping")
	require.NoError(t, err, "cannot ping")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "wrong status")

	var reply PingResponse
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&reply))
	require.Equal(t, "ok", reply.Replies["words"], "no words running")
	require.Equal(t, "ok", reply.Replies["update"], "no update running")
	require.Equal(t, "ok", reply.Replies["search"], "no search running")
}

type UpdateStats struct {
	WordsTotal    int `json:"words_total"`
	WordsUnique   int `json:"words_unique"`
	ComicsFetched int `json:"comics_fetched"`
	ComicsTotal   int `json:"comics_total"`
}

type UpdateStatus struct {
	Status string `json:"status"`
}

func login(t *testing.T) string {
	data := bytes.NewBufferString(`{"name":"admin", "password":"password"}`)
	req, err := http.NewRequest(http.MethodPost, address+"/api/login", data)
	require.NoError(t, err, "cannot make request")
	resp, err := client.Do(req)
	require.NoError(t, err, "could not send login command")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	token, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(token)
}

func prepare(t *testing.T) {
	req, err := http.NewRequest(http.MethodDelete, address+"/api/db", nil)
	require.NoError(t, err, "cannot make request")
	token := login(t)
	req.Header.Add("Authorization", "Token "+token)
	resp, err := client.Do(req)
	require.NoError(t, err, "could not send clean up command")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	st := stats(t)
	require.Equal(t, 0, st.ComicsFetched)
	require.True(t, st.ComicsTotal > 3000, "there are more than 3000 comics in XKCD")
	require.Equal(t, 0, st.WordsTotal)
	require.Equal(t, 0, st.WordsUnique)

	require.Equal(t, "idle", status(t))
}

func update(t *testing.T) int {
	req, err := http.NewRequest(http.MethodPost, address+"/api/db/update", nil)
	require.NoError(t, err, "cannot make request")
	token := login(t)
	req.Header.Add("Authorization", "Token "+token)
	resp, err := client.Do(req)
	require.NoError(t, err, "could not send update command")
	defer resp.Body.Close()
	return resp.StatusCode
}

func status(t *testing.T) string {
	resp, err := client.Get(address + "/api/db/status")
	require.NoError(t, err, "could not get status")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var status UpdateStatus
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&status), "cannot decode")
	return status.Status
}

func stats(t *testing.T) UpdateStats {
	resp, err := client.Get(address + "/api/db/stats")
	require.NoError(t, err, "could not get stats")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	var stats UpdateStats
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&stats), "cannot decode")
	return stats
}

func TestEmptyDB(t *testing.T) {
	prepare(t)
}

func TestUpdate(t *testing.T) {
	prepare(t)
	var wg sync.WaitGroup
	wg.Add(3)
	var res1, res2 int
	var res3 string
	go func() {
		res1 = update(t)
		wg.Done()
	}()
	go func() {
		res2 = update(t)
		wg.Done()
	}()
	go func() {
		time.Sleep(1 * time.Second)
		res3 = status(t)
		wg.Done()
	}()
	wg.Wait()
	require.True(t,
		res1 == http.StatusOK && res2 == http.StatusAccepted ||
			res2 == http.StatusOK && res1 == http.StatusAccepted,
		"wrong statuses from concurrent updates, expect ok && accepted",
	)
	require.Equal(t, "running", res3, "need running status while update")
	st := stats(t)
	require.Equal(t, st.ComicsTotal, st.ComicsFetched)
	require.True(t, st.ComicsTotal > 3000, "there are more than 3000 comics in XKCD")
	require.True(t, 1000 < st.WordsTotal, "not enough total words in DB")
	require.True(t, 100 < st.WordsUnique, "not enough unique words in DB")
}

type Comics struct {
	ID  int    `json:"id"`
	URL string `json:"url"`
}

type ComicsReply struct {
	Comics []Comics `json:"comics"`
	Total  int      `json:"total"`
}

func TestSearchNoPhrase(t *testing.T) {
	resp, err := client.Get(address + "/api/search")
	require.NoError(t, err, "failed to search")
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "need bad request")

	resp, err = client.Get(address + "/api/isearch")
	require.NoError(t, err, "failed to search")
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "need bad request")
}

func TestSearchBadLimitMinus(t *testing.T) {
	resp, err := client.Get(address + "/api/search?limit=-1")
	require.NoError(t, err, "failed to search")
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "need bad request")

	resp, err = client.Get(address + "/api/isearch?limit=-1")
	require.NoError(t, err, "failed to search")
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "need bad request")
}

func TestSearchBadLimitAlpha(t *testing.T) {
	resp, err := client.Get(address + "/api/search?limit=asdf")
	require.NoError(t, err, "failed to search")
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "need bad request")

	resp, err = client.Get(address + "/api/isearch?limit=asdf")
	require.NoError(t, err, "failed to search")
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "need bad request")
}

func TestSearchLimit2(t *testing.T) {
	update(t)
	resp, err := client.Get(address + "/api/search?limit=2&phrase=linux")
	require.NoError(t, err, "failed to search")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "need OK status")
	var comics ComicsReply
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&comics), "decode failed")
	require.Equal(t, 2, comics.Total)
	require.Equal(t, 2, len(comics.Comics))
}

func TestSearchLimitDefault(t *testing.T) {
	update(t)
	resp, err := client.Get(address + "/api/search?phrase=linux")
	require.NoError(t, err, "failed to search")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "need OK status")
	var comics ComicsReply
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&comics), "decode failed")
	require.Equal(t, 10, comics.Total)
	require.Equal(t, 10, len(comics.Comics))
}

func TestSearchPhrases(t *testing.T) {
	update(t)
	testCases := []struct {
		phrase string
		url    string
	}{
		{
			phrase: "linux+cpu+video+machine+русские+хакеры",
			url:    "https://imgs.xkcd.com/comics/supported_features.png",
		},
		{
			phrase: "Binary Christmas Tree",
			url:    "https://imgs.xkcd.com/comics/tree.png",
		},
		{
			phrase: "apple a day -> keeps doctors away",
			url:    "https://imgs.xkcd.com/comics/an_apple_a_day.png",
		},
		{
			phrase: "mines, captcha",
			url:    "https://imgs.xkcd.com/comics/mine_captcha.png",
		},
		{
			phrase: "newton apple's idea",
			url:    "https://imgs.xkcd.com/comics/inspiration.png",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.phrase, func(t *testing.T) {
			resp, err := client.Get(address + "/api/search?phrase=" + url.QueryEscape(tc.phrase))
			require.NoError(t, err, "failed to search")
			defer resp.Body.Close()
			require.Equal(t, http.StatusOK, resp.StatusCode, "need OK status")
			var comics ComicsReply
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&comics), "decode failed")
			urls := make([]string, 0, len(comics.Comics))
			for _, c := range comics.Comics {
				urls = append(urls, c.URL)
			}
			require.Containsf(t, urls, tc.url, "could not find %q", tc.phrase)
		})
	}
}

func TestIndexSearchPhrasesLongTest(t *testing.T) {
	prepare(t)
	time.Sleep(30 * time.Second)
	resp, err := client.Get(address + "/api/isearch?phrase=linux")
	require.NoError(t, err, "failed to search")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "need OK status")
	var comics ComicsReply
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&comics), "decode failed")
	require.Equal(t, 0, comics.Total)
	require.Equal(t, 0, len(comics.Comics))
	update(t)
	time.Sleep(30 * time.Second)

	testCases := []struct {
		phrase string
		url    string
	}{
		{
			phrase: "linux+cpu+video+machine+русские+хакеры",
			url:    "https://imgs.xkcd.com/comics/supported_features.png",
		},
		{
			phrase: "Binary Christmas Tree",
			url:    "https://imgs.xkcd.com/comics/tree.png",
		},
		{
			phrase: "apple a day -> keeps doctors away",
			url:    "https://imgs.xkcd.com/comics/an_apple_a_day.png",
		},
		{
			phrase: "mines, captcha",
			url:    "https://imgs.xkcd.com/comics/mine_captcha.png",
		},
		{
			phrase: "newton apple's idea",
			url:    "https://imgs.xkcd.com/comics/inspiration.png",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.phrase, func(t *testing.T) {
			resp, err := client.Get(address + "/api/isearch?phrase=" + url.QueryEscape(tc.phrase))
			require.NoError(t, err, "failed to search")
			defer resp.Body.Close()
			require.Equal(t, http.StatusOK, resp.StatusCode, "need OK status")
			var comics ComicsReply
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&comics), "decode failed")
			urls := make([]string, 0, len(comics.Comics))
			for _, c := range comics.Comics {
				urls = append(urls, c.URL)
			}
			require.Containsf(t, urls, tc.url, "could not find %q", tc.phrase)
		})
	}
}

// 200 tests in packs of 20, with concurrency 10. 100 reqs must be ok, the rest - 503
func TestSearchConcurrency(t *testing.T) {
	const numPacks = 10
	const packSize = 20
	const concurrency = 10
	update(t)
	var countOK atomic.Int64
	var countBusy atomic.Int64
	for range numPacks {
		var wg sync.WaitGroup
		wg.Add(packSize)
		for range packSize {
			go func() {
				defer wg.Done()
				resp, err := client.Get(address + "/api/search?phrase=linux")
				require.NoError(t, err, "failed to search")
				defer resp.Body.Close()
				switch resp.StatusCode {
				case http.StatusServiceUnavailable:
					countBusy.Add(1)
				case http.StatusOK:
					countOK.Add(1)
				}
			}()
		}
		wg.Wait()
	}
	require.True(t, int64(concurrency*numPacks) <= countOK.Load(), "need some http ok")
	require.True(t, int64(0) < countBusy.Load(), "need at least some http busy")
	require.Equal(t, int64(numPacks*packSize), countOK.Load()+countBusy.Load(),
		"need only ok and busy statuses")
}

func TestSearchRateLong(t *testing.T) {
	const rate = 100
	const numReq = 1000
	update(t)
	time.Sleep(30 * time.Second)
	var wg sync.WaitGroup
	wg.Add(numReq)
	start := time.Now()
	for range numReq {
		go func() {
			defer wg.Done()
			resp, err := client.Get(address + "/api/isearch?phrase=linux")
			require.NoError(t, err, "failed to search")
			defer resp.Body.Close()
			require.Equal(t, http.StatusOK, resp.StatusCode)
		}()
	}
	wg.Wait()
	duration := time.Since(start)
	actualRate := numReq / duration.Seconds()

	require.InDelta(t, rate, actualRate, rate/10)
}

func TestBadLogin(t *testing.T) {
	data := bytes.NewBufferString(`{"name":"user", "password":""}`)
	req, err := http.NewRequest(http.MethodPost, address+"/api/login", data)
	require.NoError(t, err, "cannot make request")
	resp, err := client.Do(req)
	require.NoError(t, err, "could not send login command")
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestBadPassword(t *testing.T) {
	data := bytes.NewBufferString(`{"name":"admin", "password":""}`)
	req, err := http.NewRequest(http.MethodPost, address+"/api/login", data)
	require.NoError(t, err, "cannot make request")
	resp, err := client.Do(req)
	require.NoError(t, err, "could not send login command")
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestGoodLogin(t *testing.T) {
	data := bytes.NewBufferString(`{"name":"admin", "password":"password"}`)
	req, err := http.NewRequest(http.MethodPost, address+"/api/login", data)
	require.NoError(t, err, "cannot make request")
	resp, err := client.Do(req)
	require.NoError(t, err, "could not send login command")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	token, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.True(t, len(token) > 0)
}

func TestLoginExpiredVeryLong(t *testing.T) {
	token := login(t)
	time.Sleep(125 * time.Second)
	req, err := http.NewRequest(http.MethodPost, address+"/api/db/update", nil)
	require.NoError(t, err, "cannot make request")
	req.Header.Add("Authorization", "Token "+token)
	resp, err := client.Do(req)
	require.NoError(t, err, "could not send update command")
	defer resp.Body.Close()
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
