package crawler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/bytedance/sonic"
	"github.com/mmcdole/gofeed"
	"github.com/tidwall/gjson"

	"github.com/flowline-io/flowbot/pkg/flog"
)

// RulePage holds page crawling configuration.
type RulePage struct {
	URL  string
	List string
	Item map[string]string
}

// RuleJson holds JSON crawling configuration.
type RuleJson struct {
	URL  string
	List string
	Item map[string]string
}

// RuleFeed holds feed crawling configuration.
type RuleFeed struct {
	URL  string
	Item map[string]string
}

type Rule struct {
	Name   string
	Enable bool
	Id     string
	When   string
	Mode   string
	Page   *RulePage `json:"page,omitempty"`
	Json   *RuleJson `json:"json,omitempty"`
	Feed   *RuleFeed `json:"feed,omitempty"`
}

func (r Rule) Run() []map[string]string {
	var result []map[string]string
	if !runPageCrawling(r, &result) {
		return result
	}
	if !runJsonCrawling(r, &result) {
		return result
	}
	runFeedCrawling(r, &result)
	return result
}

func runPageCrawling(r Rule, result *[]map[string]string) bool {
	if r.Page == nil {
		return true
	}
	doc, err := document(r.Page.URL)
	if err != nil {
		flog.Error(err)
		return false
	}

	keys := make([]string, 0, len(r.Page.Item))
	for k := range r.Page.Item {
		keys = append(keys, k)
	}
	doc.Find(r.Page.List).Each(func(_ int, s *goquery.Selection) {
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
		*result = append(*result, tmp)
	})
	return true
}

func runJsonCrawling(r Rule, result *[]map[string]string) bool {
	if r.Json == nil {
		return true
	}
	doc, err := document(r.Json.URL)
	if err != nil {
		return false
	}

	gjson.AddModifier("expand", func(raw, arg string) string {
		var args map[string]string
		err := sonic.Unmarshal([]byte(arg), &args)
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
		*result = append(*result, tmp)
	}
	return true
}

func feedItemValue(item *gofeed.Item, selector string) string {
	switch selector {
	case "guid":
		return item.GUID
	case "title":
		return item.Title
	case "description":
		return item.Description
	case "content":
		return item.Content
	case "link":
		return item.Link
	case "updated":
		return item.Updated
	case "published":
		return item.Published
	case "enclosure":
		if len(item.Enclosures) > 0 {
			return item.Enclosures[0].URL
		}
	}
	return ""
}

func runFeedCrawling(r Rule, result *[]map[string]string) {
	if r.Feed == nil {
		return
	}
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(r.Feed.URL)
	if err != nil {
		return
	}

	keys := make([]string, 0, len(r.Feed.Item))
	for k := range r.Feed.Item {
		keys = append(keys, k)
	}

	for _, item := range feed.Items {
		tmp := make(map[string]string)
		for _, k := range keys {
			v := feedItemValue(item, r.Feed.Item[k])
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
		*result = append(*result, tmp)
	}
}

type Result struct {
	Name   string
	ID     string
	Mode   string
	Result []map[string]string
}

func document(url string) (*goquery.Document, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to new request: %w", err)
	}

	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}

	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to do request: %d", res.StatusCode)
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}
	flog.Info("[crawler] Get %s content length: %d", url, len(b))

	return goquery.NewDocumentFromReader(bytes.NewReader(b))
}
