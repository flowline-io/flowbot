package version

const (
	// CurrentVersion is the current API/protocol version
	CurrentVersion = "0.1"
	// MinSupportedVersion is the minimum supported API version
	MinSupportedVersion = "0.1"
)

// Buildstamp version number defined by the compiler:
//
//	-ldflags "-X main.buildstamp=value_to_assign_to_buildstamp"
//
// Reported to clients in response to {hi} message.
// For instance, to define the buildstamp as a timestamp of when the server was built add a
// flag to compiler command line:
//
//	-ldflags "-X main.buildstamp=`date -u '+%Y%m%dT%H:%M:%SZ'`"
//
// or to set it to git tag:
//
//	-ldflags "-X main.buildstamp=`git describe --tags`"
var Buildstamp = "undef"
