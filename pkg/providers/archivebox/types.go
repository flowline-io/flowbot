package archivebox

type Data struct {
	Urls       []string `json:"urls"`
	Tag        string   `json:"tag"`
	Depth      int      `json:"depth"`
	Update     bool     `json:"update"`
	UpdateAll  bool     `json:"update_all"`
	IndexOnly  bool     `json:"index_only"`
	Overwrite  bool     `json:"overwrite"`
	Init       bool     `json:"init"`
	Extractors string   `json:"extractors"`
	Parser     string   `json:"parser"`
}

type Response struct {
	Success bool     `json:"success"`
	Errors  []string `json:"errors"`
	Result  []string `json:"result"`
	Stdout  string   `json:"stdout"`
	Stderr  string   `json:"stderr"`
}
