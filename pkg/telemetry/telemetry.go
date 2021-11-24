package telemetry

import (
	"bytes"
	"encoding/json"
	"net/http"
	"runtime"

	"github.com/cloudskiff/driftctl/build"
	"github.com/cloudskiff/driftctl/pkg/memstore"
	"github.com/cloudskiff/driftctl/pkg/version"
	"github.com/sirupsen/logrus"
)

type telemetry struct {
	Version        string `json:"version"`
	Os             string `json:"os"`
	Arch           string `json:"arch"`
	TotalResources int    `json:"total_resources"`
	TotalManaged   int    `json:"total_managed"`
	Duration       uint   `json:"duration"`
	ProviderName   string `json:"provider_name"`
}

type Telemetry struct {
	build build.BuildInterface
}

func NewTelemetry(build build.BuildInterface) *Telemetry {
	return &Telemetry{build: build}
}

func (te Telemetry) SendTelemetry(store memstore.Bucket) {

	if !te.build.IsThirdPartyNetworkCallsAllowed() {
		logrus.Debug("Third party network call are disabled on this build, telemetry skipped")
		return
	}

	t := &telemetry{
		Version: version.Current(),
		Os:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}

	if val, ok := store.Get("total_resources").(int); ok {
		t.TotalResources = val
	}

	if val, ok := store.Get("total_managed").(int); ok {
		t.TotalManaged = val
	}

	if val, ok := store.Get("duration").(uint); ok {
		t.Duration = val
	}

	if val, ok := store.Get("provider_name").(string); ok {
		t.ProviderName = val
	}

	body, err := json.Marshal(t)
	if err != nil {
		logrus.Debug(err)
		return
	}

	client := &http.Client{}
	req, _ := http.NewRequest("POST", "https://2lvzgmrf2e.execute-api.eu-west-3.amazonaws.com/telemetry", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	_, err = client.Do(req)
	if err != nil {
		logrus.Debugf("Unable to send telemetry data: %+v", err)
		return
	}

}
