package shiori

type BookmarksResponse struct {
	Bookmarks []struct {
		Id         int    `json:"id"`
		Url        string `json:"url"`
		Title      string `json:"title"`
		Excerpt    string `json:"excerpt"`
		Author     string `json:"author"`
		Public     int    `json:"public"`
		Modified   string `json:"modified"`
		ImageURL   string `json:"imageURL"`
		HasContent bool   `json:"hasContent"`
		HasArchive bool   `json:"hasArchive"`
		Tags       []struct {
			Id   int    `json:"id"`
			Name string `json:"name"`
		} `json:"tags"`
		CreateArchive bool `json:"createArchive"`
	} `json:"bookmarks"`
	MaxPage int `json:"maxPage"`
	Page    int `json:"page"`
}

type LoginResponse struct {
	Session string `json:"session"`
	Account struct {
		Id       int    `json:"id"`
		Username string `json:"username"`
		Owner    bool   `json:"owner"`
	} `json:"account"`
}

type BookmarkResponse struct {
	Id         int    `json:"id"`
	Url        string `json:"url"`
	Title      string `json:"title"`
	Excerpt    string `json:"excerpt"`
	Author     string `json:"author"`
	Public     int    `json:"public"`
	Modified   string `json:"modified"`
	Html       string `json:"html"`
	ImageURL   string `json:"imageURL"`
	HasContent bool   `json:"hasContent"`
	HasArchive bool   `json:"hasArchive"`
	Tags       []struct {
		Name string `json:"name"`
	} `json:"tags"`
	CreateArchive bool `json:"createArchive"`
}
