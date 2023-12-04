package version

const (
	// CurrentVersion is the current API/protocol version
	CurrentVersion = "0.1"
)

// Buildstamp version number defined by the compiler:
//
//	-ldflags "-X version.Buildstamp=value_to_assign_to_buildstamp"
//
// Reported to clients in response to {hi} message.
// For instance, to define the buildstamp as a timestamp of when the server was built add a
// flag to compiler command line:
//
//	-ldflags "-X version.Buildstamp=`date -u '+%Y%m%dT%H:%M:%SZ'`"
//
// or to set it to git tag:
//
//	-ldflags "-X version.Buildstamp=`git describe --tags`"
var Buildstamp = "undef"
