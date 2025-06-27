//go:build !windows && !darwin

package desktop

type Desktop struct{}

func (d Desktop) Notify(title, message string) {}

func (d Desktop) Beep() {}

func (d Desktop) Alert(title, message string) {}
