package utils

import (
	"reflect"
	"runtime"
	"strings"
)

func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func ParseFunctionName(name string) (string, string) {
	names := strings.Split(name, ".")
	if len(names) > 1 {
		return strings.Join(names[:len(names)-1], "."), names[len(names)-1]
	}
	return "", ""
}
