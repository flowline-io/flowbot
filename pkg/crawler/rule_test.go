package crawler

import "testing"

func TestHtmlRuleRun(t *testing.T) {
	var html struct {
		URL  string
		List string
		Item map[string]string
	}
	html.URL = "https://news.ycombinator.com/news"
	html.List = "tr.athing"
	html.Item = map[string]string{
		"title": `$(".title .titleline a").text`,
		"url":   `$(".title .titleline a").href`,
	}
	r := Rule{
		Name: "httpbin",
		Id:   "8zwgwc3y_2E",
		When: "* * * * *",
		Mode: "daily",
		Page: &html,
	}
	result := r.Run()
	if len(result) == 0 {
		t.Fatal("html rule run error")
	}
}

func TestJsonRuleRun(t *testing.T) {
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
