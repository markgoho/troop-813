package server_test

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/testenv"
)

func TestHealthReportsTodayInNewYork(t *testing.T) {
	t.Parallel()
	env := testenv.Setup(t)
	// 03:30 UTC on Nov 10 is still 22:30 on Monday Nov 9 in America/New_York.
	env.Clock.Set(time.Date(2026, 11, 10, 3, 30, 0, 0, time.UTC))

	resp := env.Request(t, http.MethodGet, "/api/health", "", nil)
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	var got contract.Health
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := contract.Health{Status: "ok", Today: "2026-11-09"}
	if got != want {
		t.Fatalf("health = %+v, want %+v", got, want)
	}
}
