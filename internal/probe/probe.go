package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/boreauxx/monitoring-bot.git/internal/postgres"
)

type Probe interface {
	Probe(ctx context.Context, asset *postgres.Asset) (*postgres.ProbeResult, error)
}

type errorResponse struct {
	Code   string `json:"code"`
	Error  string `json:"error"`
	Status int    `json:"status"`
}

func parseResponse(reader io.Reader) (*errorResponse, error) {
	response := new(errorResponse)

	if err := json.NewDecoder(reader).Decode(&response); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return response, nil
}
