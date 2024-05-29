package dev

import (
	"bytes"
	"crypto/rand"
	_ "embed"
	"math/big"

	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
	"gonum.org/v1/plot/plotter"
)

// randomPoints returns some random x, y points.
func randomPoints(n int) plotter.XYs {
	pts := make(plotter.XYs, n)
	for i := range pts {
		num, _ := rand.Int(rand.Reader, big.NewInt(100))
		if i == 0 {
			pts[i].X = float64(num.Int64())
		} else {
			pts[i].X = pts[i-1].X + float64(num.Int64())
		}
		pts[i].Y = pts[i].X + 10*float64(num.Int64())
	}
	return pts
}

func qrEncode(text string) (types.MsgPayload) {
	qrc, err := qrcode.New(text)
	if err != nil {
		flog.Error(err)
		return types.TextMsg{Text: err.Error()}
	}

	w := newByteWriter()
	std := standard.NewWithWriter(w)

	err = qrc.Save(std)
	if err != nil {
		flog.Error(err)
		return types.TextMsg{Text: err.Error()}
	}

	return types.ImageConvert(w.Buf.Bytes(), "QR", 200, 200)
}

type byteWriter struct {
	Buf *bytes.Buffer
}

func newByteWriter() *byteWriter {
	return &byteWriter{Buf: bytes.NewBufferString("")}
}

func (w *byteWriter) Write(p []byte) (n int, err error) {
	return w.Buf.Write(p)
}

func (w *byteWriter) Close() error {
	return nil
}
