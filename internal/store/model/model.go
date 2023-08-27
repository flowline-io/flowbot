package model

import "time"

// IsExpired check expired
func (p *Parameter) IsExpired() bool {
	return p.ExpiredAt.Before(time.Now())
}
