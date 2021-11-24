package telemetry

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"runtime"
	"testing"

	"github.com/cloudskiff/driftctl/pkg/analyser"
	"github.com/cloudskiff/driftctl/pkg/memstore"
	"github.com/cloudskiff/driftctl/pkg/resource"
	"github.com/cloudskiff/driftctl/pkg/version"
	"github.com/cloudskiff/driftctl/test/mocks"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestSendTelemetry(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	tests := []struct {
		name           string
		analysis       *analyser.Analysis
		expectedBody   *telemetry
		response       *http.Response
		setStoreValues func(memstore.Bucket, *analyser.Analysis)
	}{
		{
			name: "valid analysis",
			analysis: func() *analyser.Analysis {
				a := &analyser.Analysis{}
				a.AddManaged(&resource.Resource{})
				a.AddUnmanaged(&resource.Resource{})
				a.Duration = 123.4 * 1e9 // 123.4 seconds
				return a
			}(),
			expectedBody: &telemetry{
				Version:        version.Current(),
				Os:             runtime.GOOS,
				Arch:           runtime.GOARCH,
				TotalResources: 2,
				TotalManaged:   1,
				Duration:       123,
				ProviderName:   "aws",
			},
			setStoreValues: func(s memstore.Bucket, a *analyser.Analysis) {
				s.Set("total_resources", a.Summary().TotalResources)
				s.Set("total_managed", a.Summary().TotalManaged)
				s.Set("duration", uint(a.Duration.Seconds()+0.5))
				s.Set("provider_name", "aws")
			},
		},
		{
			name: "valid analysis with round up",
			analysis: func() *analyser.Analysis {
				a := &analyser.Analysis{}
				a.Duration = 123.5 * 1e9 // 123.5 seconds
				return a
			}(),
			expectedBody: &telemetry{
				Version:      version.Current(),
				Os:           runtime.GOOS,
				Arch:         runtime.GOARCH,
				Duration:     124,
				ProviderName: "aws",
			},
			setStoreValues: func(s memstore.Bucket, a *analyser.Analysis) {
				s.Set("total_resources", a.Summary().TotalResources)
				s.Set("total_managed", a.Summary().TotalManaged)
				s.Set("duration", uint(a.Duration.Seconds()+0.5))
				s.Set("provider_name", "aws")
			},
		},
		{
			name:     "nil analysis",
			analysis: nil,
		},
		{
			name: "incomplete analysis values",
			analysis: func() *analyser.Analysis {
				a := &analyser.Analysis{}
				a.Duration = 123.5 * 1e9 // 123.5 seconds
				return a
			}(),
			expectedBody: &telemetry{
				Version: version.Current(),
				Os:      runtime.GOOS,
				Arch:    runtime.GOARCH,
			},
			setStoreValues: func(s memstore.Bucket, a *analyser.Analysis) {},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := memstore.New().Bucket(memstore.TelemetryBucket)

			if tt.analysis != nil {
				tt.setStoreValues(store, tt.analysis)
			}

			httpmock.Reset()
			if tt.expectedBody != nil {
				httpmock.RegisterResponder(
					"POST",
					"https://2lvzgmrf2e.execute-api.eu-west-3.amazonaws.com/telemetry",
					func(req *http.Request) (*http.Response, error) {

						requestTelemetry := &telemetry{}
						requestBody, err := ioutil.ReadAll(req.Body)
						if err != nil {
							t.Fatal(err)
						}
						err = json.Unmarshal(requestBody, requestTelemetry)
						if err != nil {
							t.Fatal(err)
						}

						assert.Equal(t, tt.expectedBody, requestTelemetry)

						response := tt.response
						if response == nil {
							response = httpmock.NewBytesResponse(202, []byte{})
						}
						return response, nil
					},
				)
			}
			tl := NewTelemetry(mocks.MockBuild{ThirdPartyNetworkCallsAllowed: true})
			tl.SendTelemetry(store)
		})
	}
}

func TestTelemetryNotSend(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	store := memstore.New().Bucket(memstore.TelemetryBucket)

	httpmock.RegisterResponder(
		"POST",
		"https://2lvzgmrf2e.execute-api.eu-west-3.amazonaws.com/telemetry",
		httpmock.NewErrorResponder(nil),
	)
	tl := NewTelemetry(mocks.MockBuild{ThirdPartyNetworkCallsAllowed: false})
	tl.SendTelemetry(store)

	assert.Zero(t, httpmock.GetTotalCallCount())
}
