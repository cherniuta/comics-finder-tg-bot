package rest

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"yadro.com/course/api/core"
)

type PingResponse struct {
	Replies map[string]string `json:"replies"`
}

func NewPingHandler(log *slog.Logger, pingers map[string]core.Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var response PingResponse
		response.Replies = make(map[string]string)

		for name, address := range pingers {
			err := address.Ping(r.Context())
			if err == nil {
				response.Replies[name] = "ok"

			} else {
				response.Replies[name] = "unavailable"
			}

		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("cannot encode reply", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	}
}

type Authenticator interface {
	Login(user, password string) (string, error)
}

type Login struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}

func NewLoginHandler(log *slog.Logger, auth Authenticator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var login Login
		if err := json.NewDecoder(r.Body).Decode(&login); err != nil {
			log.Error("could not decode login form", "error", err)
			http.Error(w, "could not parse login data", http.StatusBadRequest)
			return
		}

		t, err := auth.Login(login.Name, login.Password)
		if err != nil {
			if errors.Is(err, core.ErrBadCredentials) {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if _, err := w.Write([]byte(t)); err != nil {
			log.Error("cannot encode reply", "error", err)
		}

	}
}

func NewUpdateHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		err := updater.Update(r.Context())
		if err != nil {
			if errors.Is(err, core.ErrAlreadyExists) {

				http.Error(w, err.Error(), http.StatusAccepted)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return

		}

	}
}

type StatsResponse struct {
	WordsTotal    int `json:"words_total"`
	WordsUnique   int `json:"words_unique"`
	ComicsFetched int `json:"comics_fetched"`
	ComicsTotal   int `json:"comics_total"`
}

func NewUpdateStatsHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := updater.Stats(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := StatsResponse{
			WordsTotal:    res.WordsTotal,
			WordsUnique:   res.WordsUnique,
			ComicsFetched: res.ComicsFetched,
			ComicsTotal:   res.ComicsTotal}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("cannot encode reply", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
	}

}

type StatusResponse struct {
	Status string `json:"status"`
}

func NewUpdateStatusHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, err := updater.Status(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		response := StatusResponse{Status: string(res)}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("cannot encode reply", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	}
}

func NewDropHandler(log *slog.Logger, updater core.Updater) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := updater.Drop(r.Context())
		if err != nil {
			log.Error("problems with the deleting data", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}

	}
}

type Comics struct {
	Id  int    `json:"id"`
	Url string `json:"url"`
}
type SearchResponse struct {
	Comics []Comics `json:"comics"`
	Total  int      `json:"total"`
}

const defaultLimit = 10

func NewSearchHandler(log *slog.Logger, searcher core.Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			limit int
			err   error
		)
		limitStr := r.URL.Query().Get("limit")
		if limitStr != "" {
			limit, err = strconv.Atoi(limitStr)
			if err != nil {
				log.Error("wrong limit", "value", limitStr)
				http.Error(w, "bad limit", http.StatusBadRequest)
				return
			}
			if limit < 0 {
				log.Error("wrong limit", "value", limitStr)
				http.Error(w, "bad limit", http.StatusBadRequest)
				return
			}

		} else {
			limit = defaultLimit
		}
		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			log.Error("no phrase")
			http.Error(w, "no phrase", http.StatusBadRequest)
			return
		}
		comics, err := searcher.Search(r.Context(), phrase, limit)
		if err != nil {
			if errors.Is(err, core.ErrNotFound) {
				http.Error(w, "no comics found", http.StatusNotFound)
				return
			}
			log.Error("problems finding comics", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := SearchResponse{
			Comics: make([]Comics, 0),
			Total:  len(comics),
		}

		for _, item := range comics {
			response.Comics = append(response.Comics, Comics{Id: item.ID, Url: item.URL})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("cannot encode reply", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	}
}

func NewSearchIndexHandler(log *slog.Logger, searcher core.Searcher) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var (
			limit int
			err   error
		)
		limitStr := r.URL.Query().Get("limit")
		if limitStr != "" {
			limit, err = strconv.Atoi(limitStr)
			if err != nil {
				log.Error("wrong limit", "value", limitStr)
				http.Error(w, "bad limit", http.StatusBadRequest)
				return
			}
			if limit < 0 {
				log.Error("wrong limit", "value", limitStr)
				http.Error(w, "bad limit", http.StatusBadRequest)
				return
			}

		} else {
			limit = defaultLimit
		}
		phrase := r.URL.Query().Get("phrase")
		if phrase == "" {
			log.Error("no phrase")
			http.Error(w, "no phrase", http.StatusBadRequest)
			return
		}
		comics, err := searcher.Search(r.Context(), phrase, limit)
		if err != nil {
			if errors.Is(err, core.ErrNotFound) {
				http.Error(w, "no comics found", http.StatusNotFound)
				return
			}
			log.Error("problems finding comics", "error", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		response := SearchResponse{
			Comics: make([]Comics, 0),
			Total:  len(comics),
		}

		for _, item := range comics {
			response.Comics = append(response.Comics, Comics{Id: item.ID, Url: item.URL})
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Error("cannot encode reply", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

	}
}
