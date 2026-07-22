package partials

import (
	"context"
	"fmt"
	"io"

	"github.com/a-h/templ"
)

// WriteTableEmptyOOB renders an empty-state row that HTMX swaps into a tbody.
func WriteTableEmptyOOB(ctx context.Context, w io.Writer, id, rowsSelector, colspan string, empty templ.Component) error {
	if _, err := fmt.Fprintf(w, `<tr id="%s" hx-swap-oob="innerHTML:%s"><td colspan="%s" class="p-0">`, id, rowsSelector, colspan); err != nil {
		return err
	}
	if err := empty.Render(ctx, w); err != nil {
		return err
	}
	_, err := io.WriteString(w, `</td></tr>`)
	return err
}
