package qr

import (
	"bytes"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/logs"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

func qrEncode(text string) (types.MsgPayload, error) {
	qrc, err := qrcode.New(text)
	if err != nil {
		logs.Err.Println(err)
		return nil, err
	}

	w := newByteWriter()
	std := standard.NewWithWriter(w)

	err = qrc.Save(std)
	if err != nil {
		logs.Err.Println(err)
	}

	return types.ImageConvert(w.Buf.Bytes(), "QR", 200, 200), nil
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
