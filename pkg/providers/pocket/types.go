package pocket

type CodeResponse struct {
	Code  string `json:"code"`
	State string `json:"state"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	Username    string `json:"username"`
}

type ListResponse struct {
	Status int             `json:"status"`
	List   map[string]Item `json:"list"`
}

type Item struct {
	Id            string `json:"item_id"`
	ResolvedId    string `json:"resolved_id"`
	GivenUrl      string `json:"given_url"`
	GivenTitle    string `json:"given_title"`
	Favorite      string `json:"favorite"`
	Status        string `json:"status"`
	TimeAdded     string `json:"time_added"`
	TimeUpdated   string `json:"time_updated"`
	TimeRead      string `json:"time_read"`
	TimeFavorited string `json:"time_favorited"`
	ResolvedTitle string `json:"resolved_title"`
	ResolvedUrl   string `json:"resolved_url"`
	Excerpt       string `json:"excerpt"`
	IsArticle     string `json:"is_article"`
	IsIndex       string `json:"is_index"`
	HasVideo      string `json:"has_video"`
	HasImage      string `json:"has_image"`
	WordCount     string `json:"word_count"`
}

type ItemResponse struct {
	Status int  `json:"status"`
	Item   Item `json:"item"`
}
