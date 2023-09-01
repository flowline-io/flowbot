package attendance

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"github.com/flowline-io/flowbot/internal/ruleset/command"
	"github.com/flowline-io/flowbot/internal/store"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/parser"
	"github.com/nikolaydubina/calendarheatmap/charts"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"image/color"
	"strings"
	"time"
)

var commandRules = []command.Rule{
	{
		Define: "info",
		Help:   `Bot info`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			return nil
		},
	},
	{
		Define: "today",
		Help:   `today detail`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			today := time.Now().Format("2006-01-02")
			datas, err := store.Chatbot.DataList(ctx.AsUser, ctx.Original, types.DataFilter{Prefix: &today})
			if err != nil || len(datas) == 0 {
				return types.TextMsg{Text: "Empty"}
			}

			topics := make(map[string]int)
			for _, data := range datas {
				keys := strings.Split(data.Key, ":")
				if len(keys) != 3 {
					continue
				}
				topic := keys[1]
				topics[topic] += 1
			}

			var texts []string
			for topic, num := range topics {
				texts = append(texts, fmt.Sprintf("%s (%d)", topic, num))
			}
			return types.TextListMsg{Texts: texts}
		},
	},
	{
		Define: "check [string] [string]",
		Help:   `punch in [topic] [summary]`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			topic, _ := tokens[1].Value.String()
			summary, _ := tokens[2].Value.String()
			today := time.Now().Format("2006-01-02")
			key := fmt.Sprintf("%s:%s:%d", today, topic, time.Now().Unix())
			var value = types.KV{"summary": summary}
			err := store.Chatbot.DataSet(ctx.AsUser, ctx.Original, key, value)
			if err != nil {
				return types.TextMsg{Text: "error"}
			}
			return types.TextMsg{Text: "ok"}
		},
	},
	{
		Define: "heatmap [string]",
		Help:   `heatmap last year [topic]`,
		Handler: func(ctx types.Context, tokens []*parser.Token) types.MsgPayload {
			topic, _ := tokens[1].Value.String()

			start := time.Now().AddDate(-1, 0, 0)
			datas, err := store.Chatbot.DataList(ctx.AsUser, ctx.Original, types.DataFilter{CreatedStart: &start})
			if err != nil || len(datas) == 0 {
				return types.TextMsg{Text: "Empty"}
			}

			// data
			counts := make(map[string]int)
			for _, data := range datas {
				keys := strings.Split(data.Key, ":")
				if len(keys) != 3 {
					continue
				}
				if keys[1] != topic {
					continue
				}
				counts[keys[0]] += 1
			}

			// heatmap
			fontFace, err := charts.LoadFontFace(defaultFontFaceBytes, opentype.FaceOptions{
				Size:    26,
				DPI:     280,
				Hinting: font.HintingNone,
			})
			if err != nil {
				flog.Error(err)
				return nil
			}

			var colorscale charts.BasicColorScale
			colorscale, err = charts.NewBasicColorscaleFromCSV(bytes.NewBuffer(defaultColorScaleBytes))
			if err != nil {
				flog.Error(err)
				return nil
			}

			conf := charts.HeatmapConfig{
				Counts:              counts,
				ColorScale:          colorscale,
				DrawMonthSeparator:  true,
				DrawLabels:          true,
				Margin:              30,
				BoxSize:             150,
				MonthSeparatorWidth: 5,
				MonthLabelYOffset:   50,
				TextWidthLeft:       300,
				TextHeightTop:       200,
				TextColor:           color.RGBA{100, 100, 100, 255},
				BorderColor:         color.RGBA{200, 200, 200, 255},
				Locale:              "en_US",
				Format:              "png",
				FontFace:            fontFace,
				ShowWeekdays: map[time.Weekday]bool{
					time.Monday:    true,
					time.Wednesday: true,
					time.Friday:    true,
				},
			}
			w := bytes.NewBufferString("")
			_ = charts.WriteHeatmap(conf, w)

			raw := base64.StdEncoding.EncodeToString(w.Bytes())

			return types.ImageMsg{
				Width:       1858,
				Height:      275,
				Alt:         "Heatmap.png",
				Mime:        "image/png",
				Size:        w.Len(),
				ImageBase64: raw,
			}
		},
	},
}
