package crawler

import (
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
	"github.com/tidwall/gjson"
	"net/http"
	"regexp"
	"strings"
)

type Rule struct {
	Name string
	Id   string
	When string
	Mode string
	Page *struct {
		URL  string
		List string
		Item map[string]string
	} `json:"page,omitempty"`
	Json *struct {
		URL  string
		List string
		Item map[string]string
	} `json:"json,omitempty"`
	Feed *struct {
		URL  string
		Item map[string]string
	} `json:"feed,omitempty"`
}

func (r Rule) Run() []map[string]string {
	var result []map[string]string

	// html
	if r.Page != nil {
		doc, err := document(r.Page.URL)
		if err != nil {
			return result
		}

		keys := make([]string, 0, len(r.Page.Item))
		for k := range r.Page.Item {
			keys = append(keys, k)
		}
		doc.Find(r.Page.List).Each(func(i int, s *goquery.Selection) {
			tmp := make(map[string]string)
			for _, k := range keys {
				f := ParseFun(s, r.Page.Item[k])
				v, err := f.Invoke()
				if err != nil {
					continue
				}
				v = strings.TrimSpace(v)
				v = strings.ReplaceAll(v, "\n", "")
				v = strings.ReplaceAll(v, "\r\n", "")
				if v == "" {
					continue
				}
				tmp[k] = v
			}
			if len(tmp) == 0 {
				return
			}
			result = append(result, tmp)
		})
	}

	// json
	if r.Json != nil {
		doc, err := document(r.Json.URL)
		if err != nil {
			return result
		}

		// mod func
		gjson.AddModifier("expand", func(raw, arg string) string {
			var args map[string]string
			err := json.Unmarshal([]byte(arg), &args)
			if err != nil {
				return ""
			}
			k := args["k"]
			v := args["v"]

			rx, err := regexp.Compile(k)
			if err != nil {
				return ""
			}

			src := strings.Trim(raw, "\"")
			var dst []byte
			m := rx.FindStringSubmatchIndex(src)
			s := rx.ExpandString(dst, v, src, m)

			return "\"" + string(s) + "\""
		})

		keys := make([]string, 0, len(r.Json.Item))
		for k := range r.Json.Item {
			keys = append(keys, k)
		}

		jRes := gjson.Parse(doc.Text())
		arr := jRes.Get(r.Json.List).Array()
		for _, item := range arr {
			tmp := make(map[string]string)
			for _, k := range keys {
				f := item.Get(r.Json.Item[k])
				v := f.String()
				v = strings.TrimSpace(v)
				v = strings.ReplaceAll(v, "\n", "")
				v = strings.ReplaceAll(v, "\r\n", "")
				if v == "" {
					continue
				}
				tmp[k] = v
			}
			if len(tmp) == 0 {
				continue
			}
			result = append(result, tmp)
		}
	}

	// feed
	if r.Feed != nil {
		fp := gofeed.NewParser()
		feed, err := fp.ParseURL(r.Feed.URL)
		if err != nil {
			return result
		}

		keys := make([]string, 0, len(r.Feed.Item))
		for k := range r.Feed.Item {
			keys = append(keys, k)
		}

		for _, item := range feed.Items {
			tmp := make(map[string]string)
			for _, k := range keys {
				v := ""
				switch r.Feed.Item[k] {
				case "title":
					v = item.Title
				case "description":
					v = item.Description
				case "content":
					v = item.Content
				case "link":
					v = item.Link
				case "updated":
					v = item.Updated
				}
				if v != "" {
					v = strings.TrimSpace(v)
					v = strings.ReplaceAll(v, "\n", "")
					v = strings.ReplaceAll(v, "\r\n", "")
					tmp[k] = v
				}
			}
			if len(tmp) == 0 {
				continue
			}
			result = append(result, tmp)
		}
	}

	return result
}

type Result struct {
	Name   string
	ID     string
	Mode   string
	Result []map[string]string
}

func document(url string) (*goquery.Document, error) {
	res, err := http.Get(url) // #nosec
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, err
	}

	return goquery.NewDocumentFromReader(res.Body)
}
