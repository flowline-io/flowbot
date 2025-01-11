package safeline

type Response struct {
	Data struct {
		Nodes []map[string]string `json:"nodes"`
	} `json:"data"`
	Err string `json:"err"`
	Msg string `json:"msg"`
}
