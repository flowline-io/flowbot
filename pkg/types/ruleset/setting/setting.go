package setting

import "github.com/flowline-io/flowbot/pkg/types"

type Rule []Row

type Row struct {
	Key    string
	Type   types.FormFieldType
	Title  string
	Detail string
}
