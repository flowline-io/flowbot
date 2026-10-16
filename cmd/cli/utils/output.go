package utils

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/flowline-io/flowbot/pkg/client"
	"github.com/flowline-io/flowbot/pkg/toolerr"
	"github.com/flowline-io/flowbot/pkg/types"
)

const (
	outputTable = "table"
	outputJSON  = "json"
)

// ErrConfirmationRequired is returned when a destructive command needs --yes
// and input is disabled.
var ErrConfirmationRequired = errors.New("confirmation required")

// ErrCancelled is returned when an operator declines a confirmation prompt.
// It is a CLI-only code and is not part of the MCP error vocabulary.
var ErrCancelled = errors.New("cancelled")

// IsJSON reports whether cmd requested JSON output.
// The flag may be local or inherited from a parent persistent flag.
func IsJSON(cmd *cobra.Command) bool {
	return flagString(cmd, "output") == outputJSON
}

// Yes reports whether --yes / -y is set on cmd or an ancestor.
func Yes(cmd *cobra.Command) bool {
	return flagBool(cmd, "yes")
}

// NoInput reports whether prompts must not read stdin.
// --no-input forces that mode. A non-terminal stdin does too.
// A pseudo-terminal still counts as a terminal.
func NoInput(cmd *cobra.Command) bool {
	if flagBool(cmd, "no-input") {
		return true
	}
	return !term.IsTerminal(int(os.Stdin.Fd()))
}

// ValidateOutput rejects --output values other than table and json.
func ValidateOutput(cmd *cobra.Command, _ []string) error {
	value := flagString(cmd, "output")
	if value == "" || value == outputTable || value == outputJSON {
		return nil
	}
	return types.Errorf(types.ErrInvalidArgument, "output must be table or json")
}

// Confirm runs a yes/no prompt.
// --yes proceeds. No-input mode returns ErrConfirmationRequired without reading stdin.
// Any answer other than y returns ErrCancelled.
func Confirm(cmd *cobra.Command, prompt string) error {
	if Yes(cmd) {
		return nil
	}
	if NoInput(cmd) {
		return fmt.Errorf("%w", ErrConfirmationRequired)
	}
	out := io.Writer(os.Stdout)
	in := io.Reader(os.Stdin)
	if cmd != nil {
		out = cmd.OutOrStdout()
		in = cmd.InOrStdin()
	}
	if _, err := fmt.Fprint(out, prompt); err != nil {
		return fmt.Errorf("write confirmation prompt: %w", err)
	}
	var response string
	if _, err := fmt.Fscanln(in, &response); err != nil {
		return fmt.Errorf("read confirmation: %w", err)
	}
	if response != "y" && response != "Y" {
		return fmt.Errorf("%w", ErrCancelled)
	}
	return nil
}

// Print writes the JSON success envelope when cmd is in JSON mode.
// Otherwise it writes a plain-text view of data.
// A value that is already a {data, page} object is not wrapped again.
// page is included only when non-nil.
func Print(cmd *cobra.Command, data, page any) error {
	w := io.Writer(os.Stdout)
	if cmd != nil {
		w = cmd.OutOrStdout()
	}
	if IsJSON(cmd) {
		return WriteEnvelope(w, data, page)
	}
	return WritePlain(w, data)
}

// WriteEnvelope writes {"data": ...} and page when page is non-nil.
func WriteEnvelope(w io.Writer, data, page any) error {
	payload := data
	if page == nil {
		if env, ok := asDataPage(data); ok {
			raw, err := sonic.MarshalIndent(env, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal json: %w", err)
			}
			_, err = fmt.Fprintln(w, string(raw))
			return err
		}
	}
	obj := map[string]any{"data": normalizeData(payload)}
	if page != nil {
		obj["page"] = page
	}
	raw, err := sonic.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal json: %w", err)
	}
	_, err = fmt.Fprintln(w, string(raw))
	return err
}

// WritePlain writes a line-oriented view of v for table mode.
func WritePlain(w io.Writer, v any) error {
	if env, ok := asDataPage(v); ok {
		v = env["data"]
	}
	return writePlainValue(w, plainValue(v), 0)
}

// PrintError writes a failure to w.
// JSON mode writes the error envelope and leaves the caller responsible for an empty stdout.
// Table mode writes Error: plus the original error text.
func PrintError(w io.Writer, jsonMode bool, err error) error {
	if err == nil {
		return nil
	}
	if !jsonMode {
		_, printErr := fmt.Fprintf(w, "Error: %v\n", err)
		return printErr
	}
	raw, mErr := sonic.MarshalIndent(map[string]any{"error": ErrorObject(err)}, "", "  ")
	if mErr != nil {
		_, printErr := fmt.Fprint(w, "{\"error\":{\"code\":\"internal\",\"message\":\"internal error\",\"retryable\":false,\"hint\":\"\"}}\n")
		return printErr
	}
	_, printErr := fmt.Fprintln(w, string(raw))
	return printErr
}

// ErrorObject returns the stderr error object for err.
func ErrorObject(err error) map[string]any {
	switch {
	case errors.Is(err, ErrCancelled):
		return map[string]any{
			"code":      "cancelled",
			"message":   "cancelled",
			"retryable": false,
			"hint":      "The operation was not performed.",
		}
	case errors.Is(err, ErrConfirmationRequired):
		return map[string]any{
			"code":      "confirmation_required",
			"message":   "confirmation required",
			"retryable": false,
			"hint":      "Pass --yes to proceed.",
		}
	default:
		return toolerr.ToToolError(Normalize(err)).Object()
	}
}

// ExitCode maps err onto the frozen CLI exit codes.
func ExitCode(err error) int {
	if err == nil {
		return 0
	}
	if errors.Is(err, ErrCancelled) {
		return 1
	}
	if errors.Is(err, ErrConfirmationRequired) {
		return 7
	}
	te := toolerr.ToToolError(Normalize(err))
	return toolerr.ProcessExit(te.Code, te.Retryable)
}

// Normalize converts API and cobra usage errors into domain errors so they share
// the MCP code table. Domain errors and confirmation sentinels pass through.
func Normalize(err error) error {
	if err == nil || errors.Is(err, ErrCancelled) || errors.Is(err, ErrConfirmationRequired) {
		return err
	}
	var te *types.Error
	if errors.As(err, &te) && te != nil {
		return err
	}
	var api *client.APIError
	if errors.As(err, &api) && api != nil {
		message := api.Message
		if message == "" {
			message = api.Error()
		}
		return types.WrapError(kindFromAPI(api), message, err)
	}
	if isUsageError(err) {
		return types.Errorf(types.ErrInvalidArgument, "%s", err.Error())
	}
	return err
}

func kindFromAPI(api *client.APIError) error {
	if api.RetCode == "10010" {
		if api.StatusCode == 409 {
			return types.ErrConflict
		}
		return types.ErrAlreadyExists
	}
	if kind := apiRetCode[api.RetCode]; kind != nil {
		return kind
	}
	if kind := apiStatus[api.StatusCode]; kind != nil {
		return kind
	}
	return types.ErrInternal
}

var apiRetCode = map[string]error{
	"10001": types.ErrInvalidArgument,
	"60005": types.ErrUnauthorized,
	"60007": types.ErrForbidden,
	"10009": types.ErrNotFound,
	"10011": types.ErrRateLimited,
	"10012": types.ErrUnavailable,
	"10013": types.ErrTimeout,
	"10002": types.ErrNotImplemented,
	"10014": types.ErrProvider,
}

var apiStatus = map[int]error{
	400: types.ErrInvalidArgument,
	401: types.ErrUnauthorized,
	403: types.ErrForbidden,
	404: types.ErrNotFound,
	409: types.ErrConflict,
	429: types.ErrRateLimited,
	408: types.ErrTimeout,
	504: types.ErrTimeout,
	501: types.ErrNotImplemented,
	503: types.ErrUnavailable,
}

func isUsageError(err error) bool {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "unknown command"):
		return true
	case strings.Contains(msg, "unknown flag"):
		return true
	case strings.Contains(msg, "unknown shorthand flag"):
		return true
	case strings.Contains(msg, "flag needs an argument"):
		return true
	case strings.Contains(msg, "bad flag syntax"):
		return true
	case strings.Contains(msg, "accepts ") && strings.Contains(msg, "arg"):
		return true
	case strings.HasPrefix(msg, "required flag"):
		return true
	default:
		return false
	}
}

func flagString(cmd *cobra.Command, name string) string {
	for c := cmd; c != nil; c = c.Parent() {
		if f := c.Flags().Lookup(name); f != nil {
			return f.Value.String()
		}
		if f := c.PersistentFlags().Lookup(name); f != nil {
			return f.Value.String()
		}
	}
	return ""
}

func flagBool(cmd *cobra.Command, name string) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if f := c.Flags().Lookup(name); f != nil {
			value, err := c.Flags().GetBool(name)
			return err == nil && value
		}
		if f := c.PersistentFlags().Lookup(name); f != nil {
			value, err := c.PersistentFlags().GetBool(name)
			return err == nil && value
		}
	}
	return false
}

func normalizeData(data any) any {
	if data == nil {
		return map[string]any{}
	}
	rv := reflect.ValueOf(data)
	switch rv.Kind() {
	case reflect.Pointer:
		if rv.IsNil() {
			return map[string]any{}
		}
	case reflect.Slice:
		if rv.IsNil() {
			return []any{}
		}
	}
	return data
}

func asDataPage(v any) (map[string]any, bool) {
	if v == nil {
		return nil, false
	}
	raw, err := sonic.Marshal(v)
	if err != nil {
		return nil, false
	}
	obj := map[string]any{}
	if err := sonic.Unmarshal(raw, &obj); err != nil {
		return nil, false
	}
	if _, ok := obj["data"]; !ok {
		return nil, false
	}
	for key := range obj {
		if key != "data" && key != "page" {
			return nil, false
		}
	}
	if _, hasPage := obj["page"]; hasPage {
		if obj["data"] == nil {
			obj["data"] = []any{}
		}
		if obj["page"] == nil {
			obj["page"] = map[string]any{"limit": 0, "has_more": false}
		}
	}
	return obj, true
}

func plainValue(v any) any {
	if v == nil {
		return nil
	}
	raw, err := sonic.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	var decoded any
	if err := sonic.Unmarshal(raw, &decoded); err != nil {
		return fmt.Sprint(v)
	}
	return decoded
}

func writePlainValue(w io.Writer, v any, indent int) error {
	pad := strings.Repeat("  ", indent)
	switch val := v.(type) {
	case nil:
		_, err := fmt.Fprintln(w, pad+"(none)")
		return err
	case string:
		_, err := fmt.Fprintln(w, pad+val)
		return err
	case map[string]any:
		return writePlainMap(w, val, pad, indent)
	case []any:
		if len(val) == 0 {
			_, err := fmt.Fprintln(w, pad+"(none)")
			return err
		}
		for _, item := range val {
			if _, ok := item.(map[string]any); ok {
				if err := writePlainValue(w, item, indent); err != nil {
					return err
				}
				if _, err := fmt.Fprintln(w); err != nil {
					return err
				}
				continue
			}
			if _, err := fmt.Fprintln(w, pad+scalarText(item)); err != nil {
				return err
			}
		}
		return nil
	default:
		_, err := fmt.Fprintln(w, pad+scalarText(val))
		return err
	}
}

func writePlainMap(w io.Writer, val map[string]any, pad string, indent int) error {
	if len(val) == 0 {
		_, err := fmt.Fprintln(w, pad+"(none)")
		return err
	}
	keys := make([]string, 0, len(val))
	for key := range val {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		child := val[key]
		_, nested := child.(map[string]any)
		_, list := child.([]any)
		if nested || list {
			if _, err := fmt.Fprintf(w, "%s%s:\n", pad, key); err != nil {
				return err
			}
			if err := writePlainValue(w, child, indent+1); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(w, "%s%s: %s\n", pad, key, scalarText(child)); err != nil {
			return err
		}
	}
	return nil
}

func scalarText(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	raw, err := sonic.Marshal(v)
	if err != nil {
		return fmt.Sprint(v)
	}
	return string(raw)
}
