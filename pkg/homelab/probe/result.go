package probe

import (
	"github.com/flowline-io/flowbot/pkg/homelab"
)

// ProbeResult groups capabilities discovered by probing a single app.
type ProbeResult struct {
	AppName      string
	Capabilities []homelab.AppCapability
}

// ProbeMatch captures a capability matched via fingerprinting together with
// a confidence score and the name of the fingerprint that produced it.
type ProbeMatch struct {
	Capability  homelab.AppCapability
	Confidence  float64
	Fingerprint string
}

