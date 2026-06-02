package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPrometheusHandlerIncludesSecurityAndReadinessMetrics(t *testing.T) {
	t.Parallel()

	Global().Reset()
	Global().IncrementWSRejected()
	Global().IncrementWSRateLimitHits()
	Global().IncrementWSMalformed()
	Global().IncrementReadinessFailures()

	req := httptest.NewRequest(http.MethodGet, "/metrics/prometheus", nil)
	rec := httptest.NewRecorder()
	PrometheusHandler()(rec, req)

	body := rec.Body.String()
	for _, metric := range []string{
		"zenthril_ws_rejected_total 1",
		"zenthril_ws_rate_limit_hits_total 1",
		"zenthril_ws_malformed_total 1",
		"zenthril_readiness_failures_total 1",
	} {
		if !strings.Contains(body, metric) {
			t.Fatalf("prometheus body missing %q:\n%s", metric, body)
		}
	}
}
