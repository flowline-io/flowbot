// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.
// Code generated by gorm.io/gen. DO NOT EDIT.

package model

const TableNameDellog = "dellog"

// Dellog mapped from table <dellog>
type Dellog struct {
	ID         int32  `gorm:"column:id;primaryKey;autoIncrement:true" json:"id"`
	Topic      string `gorm:"column:topic;not null" json:"topic"`
	Deletedfor int64  `gorm:"column:deletedfor;not null" json:"deletedfor"`
	Delid      int32  `gorm:"column:delid;not null" json:"delid"`
	Low        int32  `gorm:"column:low;not null" json:"low"`
	Hi         int32  `gorm:"column:hi;not null" json:"hi"`
}

// TableName Dellog's table name
func (*Dellog) TableName() string {
	return TableNameDellog
}
