// Package api provides the HTTP API layer for NetPulse.
package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/vosskstudio/netpulse/internal/model"
)

func writeJSON(w http.ResponseWriter, data interface{}, err error, elapsed time.Duration) {
	w.Header().Set("Content-Type", "application/json")
	resp := model.APIResponse{OK: err == nil, Data: data, ElapsedMs: elapsed.Milliseconds()}
	if err != nil {
		resp.Error = err.Error()
	}
	json.NewEncoder(w).Encode(resp)
}
