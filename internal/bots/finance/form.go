package finance

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/providers/fireflyiii"
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

			// extract bill records
			ctx.SetTimeout(10 * time.Minute)
			content, err := billParser(ctx.Context(), billText)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "failed to parse bill"}
			}

			client := fireflyiii.GetClient()
			if client == nil {
				return types.TextMsg{Text: "failed to get firefly client"}
			}

			// create transaction records
			var result struct {
				Records []struct {
					Date        string  `json:"date"`
					Amount      float64 `json:"amount"`
					Merchant    string  `json:"merchant"`
					Description string  `json:"description"`
				} `json:"records"`
			}

			// Extract JSON content
			start := strings.Index(content, "{")
			end := strings.LastIndex(content, "}") + 1
			if start >= 0 && end > start {
				content = content[start:end]
			}

			err = json.Unmarshal([]byte(content), &result)
			if err != nil {
				flog.Error(err)
				return types.TextMsg{Text: "failed to parse records"}
			}

			utils.PrettyPrintJsonStyle(result)

			for _, record := range result.Records {
				// Validate date format
				_, err := time.Parse("2006-01-02 15:04:05", record.Date)
				if err != nil {
					continue
				}

				// Create transaction
				transaction := fireflyiii.Transaction{
					ApplyRules:   true,
					FireWebhooks: true,
					Transactions: []fireflyiii.TransactionRecord{
						{
							Type:            string(fireflyiii.Withdrawal),
							Date:            record.Date,
							Amount:          fmt.Sprintf("%.2f", record.Amount),
							Description:     record.Merchant,
							SourceId:        "1", // Default account ID
							SourceName:      "",
							DestinationId:   0,
							DestinationName: "",
						},
					},
				}

				transactionResult, err := client.CreateTransaction(transaction)
				if err != nil {
					flog.Error(err)
					return types.TextMsg{Text: "failed to create transactions"}
				}

				flog.Info("Successfully imported %+v", transactionResult)
			}

			return types.TextMsg{Text: "Successfully imported"}
		},
	},
}
