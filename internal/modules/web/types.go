package web

import (
	"github.com/a-h/templ"

	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/views/partials"
)

// viewTemplateFn is a function that takes the data payload from page_data
// and returns a templ component for that content type.
type viewTemplateFn func(data types.KV) templ.Component

// viewTemplates maps page_data type strings to their rendering functions.
var viewTemplates = map[string]viewTemplateFn{
	"text":         textView,
	"markdown":     markdownView,
	"image":        imageView,
	"pipeline_run": pipelineRunView,
	"form":         formView,
}

// textView renders plain text content in a <pre> block.
func textView(data types.KV) templ.Component {
	content, _ := data.String("content")
	return partials.ViewTextContent(content)
}

// markdownView renders markdown content.
func markdownView(data types.KV) templ.Component {
	content, _ := data.String("content")
	return partials.ViewMarkdownContent(content)
}

// imageView renders an image.
func imageView(data types.KV) templ.Component {
	url, _ := data.String("url")
	alt, _ := data.String("alt")
	return partials.ViewImageContent(url, alt)
}

// pipelineRunView renders pipeline step run results.
func pipelineRunView(data types.KV) templ.Component {
	steps, _ := data.Any("steps")
	return partials.ViewPipelineRunContent(steps)
}

// formView renders a read-only form with label-value pairs.
func formView(data types.KV) templ.Component {
	fields, _ := data.List("fields")
	return partials.ViewFormContent(fields)
}
