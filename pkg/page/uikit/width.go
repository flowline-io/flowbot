package uikit

import "fmt"

const (
	WidthAutoClass   = "uk-width-auto"
	WidthExpandClass = "uk-width-expand"
)

func WidthClass(i, j int) string {
	return fmt.Sprintf("uk-width-%d-%d", i, j)
}

const (
	ChildWidthAutoClass   = "uk-child-width-auto"
	ChildWidthExpandClass = "uk-child-width-expand"
)

func ChildWidthClass(i, j int) string {
	return fmt.Sprintf("uk-child-width-%d-%d", i, j)
}
