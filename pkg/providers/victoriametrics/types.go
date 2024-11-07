package victoriametrics

type MetricsResponse struct {
	Status string `json:"status"`
	Data   Data   `json:"data"`
	Stats  Stats  `json:"stats"`
}

type Data struct {
	ResultType string   `json:"resultType"`
	Result     []Result `json:"result"`
}

type Result struct {
	Metric Metric        `json:"metric"`
	Value  []interface{} `json:"value"`
}

type Metric struct {
	Instance string `json:"instance"`
}

type Stats struct {
	SeriesFetched     string `json:"seriesFetched"`
	ExecutionTimeMsec int    `json:"executionTimeMsec"`
}
