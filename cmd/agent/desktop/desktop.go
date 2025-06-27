//go:build windows || darwin

package desktop

import (
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/gen2brain/beeep"
)

type Desktop struct{}

func (d Desktop) Notify(title, message string) {
	err := beeep.Notify(title, message, "")
	if err != nil {
		flog.Error(err)
	}
}

func (d Desktop) Beep() {
	err := beeep.Beep(beeep.DefaultFreq, beeep.DefaultDuration)
	if err != nil {
		flog.Error(err)
	}
}

func (d Desktop) Alert(title, message string) {
	err := beeep.Alert(title, message, "")
	if err != nil {
		flog.Error(err)
	}
}
