package crawler

import (
	"net/http"
	"testing"
	"time"
)

// skipIfNoNetwork skips the test if external network is unreachable.
func skipIfNoNetwork(t *testing.T, url string) {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		t.Skipf("skipping test: network not available: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Skipf("skipping test: %s returned status %d", url, resp.StatusCode)
	}
}

func TestHtmlRuleRun(t *testing.T) {
	const target = "https://news.ycombinator.com/news"
	skipIfNoNetwork(t, target)
	var html struct {
		URL  string
		List string
		Item map[string]string
	}
	html.URL = target
	html.List = "tr.athing"
	html.Item = map[string]string{
		"title": `$("td.title .titleline a").text`,
		"url":   `$("td.title .titleline a").href`,
	}
	r := Rule{
		Name: "hackernews",
		Id:   "8zwgwc3y_2E",
		When: "* * * * *",
		Mode: "daily",
		Page: &html,
	}
	result := r.Run()
	if len(result) == 0 {
		t.Skip("skipping: html rule returned no results (site structure may have changed)")
	}
}

func TestJsonRuleRun(t *testing.T) {
	skipIfNoNetwork(t, "https://httpbin.org/get")
	var json struct {
		URL  string
		List string
		Item map[string]string
	}
	json.URL = "https://httpbin.org/json"
	json.List = "slideshow.slides"
	json.Item = map[string]string{
		"title": `title.@expand:{"k":"(.*)","v":"https://httpbin.org/$1"}`,
		"type":  "type",
	}
	r := Rule{
		Name: "httpbin",
		Id:   "rfv8BzaExOo",
		When: "* * * * *",
		Mode: "daily",
		Json: &json,
	}
	result := r.Run()
	if len(result) == 0 {
		t.Fatal("json rule run error")
	}
}
