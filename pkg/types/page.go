package types

import (
	"time"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type UI struct {
	Title     string
	App       app.UI
	CSS       []app.UI
	JS        []app.HTMLScript
	Global    KV
	ExpiredAt time.Time
}
