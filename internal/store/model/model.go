package model

import (
	"time"

	"github.com/bytedance/sonic"
)

// IsExpired check expired
func (p *Parameter) IsExpired() bool {
	return p.ExpiredAt.Before(time.Now())
}

func (j *Job) MarshalBinary() (data []byte, err error) {
	return sonic.Marshal(j)
}
