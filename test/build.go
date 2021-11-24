package test

type Build struct{}

func (b Build) IsRelease() bool {
	return false
}

func (b Build) IsThirdPartyNetworkCallsAllowed() bool {
	return false
}
