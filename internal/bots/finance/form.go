package finance

import (
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/flowline-io/flowbot/pkg/ability"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/hub"
	"github.com/flowline-io/flowbot/pkg/types"
	"github.com/flowline-io/flowbot/pkg/types/ruleset/form"
	"github.com/flowline-io/flowbot/pkg/utils"
)

const (
	importBillFormID = "import_bill"
)

var formRules = []form.Rule{
	{
		Id:    importBillFormID,
		Title: "Import Bill",
		Field: []types.FormField{
			{
				Key:         "bill",
				Type:        types.FormFieldTextarea,
				ValueType:   types.FormFieldValueString,
				Value:       "",
				Label:       "Textarea",
				Placeholder: "Input bill",
				Rule:        "required",
			},
		},
		Handler: func(ctx types.Context, values types.KV) types.MsgPayload {
			billText, _ := values.String("bill")
			if billText == "" {
				return types.TextMsg{Text: "bill text is empty"}
			}

			ctx.SetTimeout(10 * time.Minute)
			content, err := billParser(ctx.Context(), billText)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "failed to parse bill"}
			}

			var result struct {
				Records []struct {
					Date        string  `json:"date"`
					Amount      float64 `json:"amount"`
					Merchant    string  `json:"merchant"`
					Description string  `json:"description"`
				} `json:"records"`
			}

			start := strings.Index(content, "{")
			end := strings.LastIndex(content, "}") + 1
			if start >= 0 && end > start {
				content = content[start:end]
			}

			err = sonic.Unmarshal([]byte(content), &result)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "failed to parse records"}
			}

			utils.PrettyPrintJsonStyle(result)

			for _, record := range result.Records {
				_, err := time.Parse("2006-01-02 15:04:05", record.Date)
				if err != nil {
					continue
				}

				res, err := ability.Invoke(ctx.Context(), hub.CapFinance, "create_transaction", map[string]any{
					"description": record.Merchant,
					"amount":      fmt.Sprintf("%.2f", record.Amount),
					"date":        record.Date,
					"source_id":   "1",
				})
				if err != nil {
					flog.Error(err)
					return types.TextMsg{Text: "failed to create transactions"}
				}

				flog.Info("Successfully imported %+v", res.Data)
			}

			return types.TextMsg{Text: "Successfully imported"}
		},
	},
}
