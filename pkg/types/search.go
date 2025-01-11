package types

type Document struct {
	Id          string `json:"id"`
	SourceId    string `json:"source_id"`
	Source      string `json:"source"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Url         string `json:"url"`
	Timestamp   int64  `json:"timestamp"`
}

type DocumentList []*Document

func (list DocumentList) FillUrlBase(urlBase map[string]string) {
	for i, item := range list {
		if v, ok := urlBase[item.Source]; ok {
			list[i].Url = v + item.Url
		}
	}
}
