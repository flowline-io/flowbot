package model

import (
	json "github.com/json-iterator/go"
	"time"
)

// IsExpired check expired
func (p *Parameter) IsExpired() bool {
	return p.ExpiredAt.Before(time.Now())
}

func (j *Job) MarshalBinary() (data []byte, err error) {
	return json.Marshal(j)
}
