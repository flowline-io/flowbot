package uikit

import "fmt"

func ColumnClass(i, j int) string {
	return fmt.Sprintf("uk-column-%d-%d", i, j)
}
