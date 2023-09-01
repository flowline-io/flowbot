package route

import (
	"bytes"
	"fmt"
	"github.com/emicklei/go-restful/v3"
	"github.com/flowline-io/flowbot/internal/types"
	"github.com/flowline-io/flowbot/pkg/flog"
	"github.com/flowline-io/flowbot/pkg/logs"
	"net/http"
	"runtime"
)

func logServiceError(serviceError restful.ServiceError, _ *restful.Request, resp *restful.Response) {
	if serviceError.Code != http.StatusNotFound {
		flog.Error(serviceError)
	}
	resp.WriteHeader(serviceError.Code)
	_ = resp.WriteAsJson(types.KV{
		"error":   serviceError.Code,
		"message": serviceError.Message,
	})
}

func logStackOnRecover(panicReason interface{}, w http.ResponseWriter) {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintf("recover from panic situation: - %v\r\n", panicReason))
	for i := 2; ; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		buffer.WriteString(fmt.Sprintf("    %s:%d\r\n", file, line))
	}
	logs.Info.Println(buffer.String())

	headers := http.Header{}
	if ct := w.Header().Get("Content-Type"); len(ct) > 0 {
		headers.Set("Accept", ct)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write(buffer.Bytes())
}
