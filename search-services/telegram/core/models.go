package core

type SearchResult struct {
	Comics []struct {
		ID  int    `json:"id"`
		URL string `json:"url"`
	} `json:"comics"`
	Total int `json:"total"`
}

type TelegramUpdate struct {
	UpdateID int64            `json:"update_id"`
	Message  *TelegramMessage `json:"message"`
}

type TelegramMessage struct {
	Chat TelegramChat `json:"chat"`
	From TelegramUser `json:"from"`
	Text string       `json:"text"`
}

type TelegramChat struct {
	ID int64 `json:"id"`
}

type TelegramUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type UserState struct {
	Step     string
	User     string
	Password string
	Limit    int
	Phrase   string
}

type TokenVerifier string

type StatsResult struct {
	WordsTotal    int
	WordsUnique   int
	ComicsFetched int
	ComicsTotal   int
}
