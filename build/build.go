package build

var env = "dev"

// This flag could be switched to false while building to create a binary without third party network calls
// That mean that following services will be disabled:
// - telemetry
// - version check
var enableThirdPartyNetworkCalls = "true"

type BuildInterface interface {
	IsRelease() bool
	IsThirdPartyNetworkCallsAllowed() bool
}

type Build struct{}

func (b Build) IsRelease() bool {
	return env == "release"
}

func (b Build) IsThirdPartyNetworkCallsAllowed() bool {
	return enableThirdPartyNetworkCalls == "true"
}
