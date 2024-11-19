package utils

import (
	"jaytaylor.com/html2text"
)

func Html2Text(html string) (string, error) {
	return html2text.FromString(html, html2text.Options{TextOnly: true})
}
