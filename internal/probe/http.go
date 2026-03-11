package probe

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/boreauxx/monitoring-bot.git/internal/postgres"
)

type HttpProbe struct{}

func (hp *HttpProbe) Probe(ctx context.Context, asset *postgres.Asset) *postgres.ProbeResult {
	probe := &postgres.ProbeResult{
		AssetID: asset.ID,
	}

	client := &http.Client{
		Timeout: time.Duration(asset.TimeoutSeconds) * time.Second,
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodHead, asset.Address, nil)
	if err != nil {
		message := fmt.Sprintf("Error creating HTTP request: %s", err)
		probe.Success = false
		probe.ErrMessage = &message
		probe.Code = http.StatusInternalServerError
		return probe
	}

	response, err := client.Do(request)
	if err != nil {
		message := fmt.Sprintf("Error sending HTTP request: %s", err)
		probe.Success = false
		probe.ErrMessage = &message
		probe.Code = http.StatusInternalServerError
		return probe
	}

	defer func() { _ = response.Body.Close() }()

	probe.Code = response.StatusCode

	if response.StatusCode >= http.StatusBadRequest {
		if body, _ := parseResponse(response.Body); body != nil {
			message := fmt.Sprintf("%s (%s)", body.Error, body.Code)
			probe.ErrMessage = &message
		}
		probe.Success = false
		return probe
	}

	probe.Success = true
	return probe
}
