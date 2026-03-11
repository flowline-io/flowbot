package utils

import (
	"time"

	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

type Debouncer struct {
	id      int64
	delay   time.Duration
	pending bool
}

func NewDebouncer(delay time.Duration) *Debouncer {
	return &Debouncer{delay: delay}
}

func (d *Debouncer) Call(ctx app.Context, fn func()) {
	d.id++
	currentID := d.id
	d.pending = true

	ctx.Async(func() {
		time.Sleep(d.delay)
		ctx.Dispatch(func(ctx app.Context) {
			if d.id != currentID {
				return
			}
			d.pending = false
			fn()
		})
	})
}

func (d *Debouncer) Pending() bool {
	return d.pending
}
