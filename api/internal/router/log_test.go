package router_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/markgoho/troop-813/api/internal/contract"
	"github.com/markgoho/troop-813/api/internal/router"
)

func TestLoggerIsNilSafe(t *testing.T) {
	t.Parallel()
	router.Logger(nil).InfoContext(context.Background(), "discarded") // must not panic
	l := slog.New(slog.DiscardHandler)
	if router.Logger(l) != l {
		t.Fatal("Logger should return a non-nil logger unchanged")
	}
}

func TestInternalErrorLogsThenAnswers500(t *testing.T) {
	t.Parallel()
	var logs bytes.Buffer
	w := httptest.NewRecorder()

	router.InternalError(context.Background(), slog.New(slog.NewJSONHandler(&logs, nil)), w,
		"kiosk check-in failed", errors.New("boom"), "meetingId", "2026-11-09")

	if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), string(contract.ErrorCodeInternal)) {
		t.Errorf("response = %d %s, want 500 internal", w.Code, w.Body)
	}
	for _, want := range []string{"kiosk check-in failed", "boom", "2026-11-09", `"level":"ERROR"`} {
		if !strings.Contains(logs.String(), want) {
			t.Errorf("log %q lacks %q", logs.String(), want)
		}
	}
}

func TestInternalErrorWithNoLoggerStillAnswers(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()

	router.InternalError(context.Background(), nil, w, "failed", errors.New("boom"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status %d, want 500", w.Code)
	}
}
