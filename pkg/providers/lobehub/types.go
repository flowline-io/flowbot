package lobehub

type WebCrawlerResponse struct {
	Content string `json:"content"`
	Title   string `json:"title"`
	Url     string `json:"url"`
	Website string `json:"website"`
}
