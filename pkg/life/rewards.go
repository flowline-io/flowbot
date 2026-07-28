package life

import "time"

// RewardOnCooldown reports whether a reward cannot be redeemed yet.
// cooldownHours <= 0 or a nil lastRedeemed means not on cooldown.
func RewardOnCooldown(lastRedeemed *time.Time, cooldownHours int, now time.Time) bool {
	if cooldownHours <= 0 || lastRedeemed == nil {
		return false
	}
	return now.Before(lastRedeemed.Add(time.Duration(cooldownHours) * time.Hour))
}

// RewardCooldownEndsAt returns when the cooldown expires, or nil if not on cooldown.
func RewardCooldownEndsAt(lastRedeemed *time.Time, cooldownHours int, now time.Time) *time.Time {
	if !RewardOnCooldown(lastRedeemed, cooldownHours, now) {
		return nil
	}
	ends := lastRedeemed.Add(time.Duration(cooldownHours) * time.Hour)
	return &ends
}
