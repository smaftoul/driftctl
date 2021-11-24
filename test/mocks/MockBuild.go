package mocks

type MockBuild struct {
	Release                       bool
	ThirdPartyNetworkCallsAllowed bool
}

func (m MockBuild) IsRelease() bool {
	return m.Release
}

func (m MockBuild) IsThirdPartyNetworkCallsAllowed() bool {
	return m.ThirdPartyNetworkCallsAllowed
}
