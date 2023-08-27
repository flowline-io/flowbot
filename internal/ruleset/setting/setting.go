package setting

import "github.com/sysatom/flowbot/internal/types"

type Rule []Row

type Row struct {
	Key    string
	Type   types.FormFieldType
	Title  string
	Detail string
}
