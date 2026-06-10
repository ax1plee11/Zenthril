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
	Global().IncrementDeviceRegistrations()
	Global().IncrementDeviceRevocations()
	Global().IncrementKeyBundleClaims()
	Global().IncrementPreKeyDepleted()

	req := httptest.NewRequest(http.MethodGet, "/metrics/prometheus", nil)
	rec := httptest.NewRecorder()
	PrometheusHandler()(rec, req)

	body := rec.Body.String()
	for _, metric := range []string{
		"zenthril_ws_rejected_total 1",
		"zenthril_ws_rate_limit_hits_total 1",
		"zenthril_ws_malformed_total 1",
		"zenthril_readiness_failures_total 1",
		"zenthril_device_registrations_total 1",
		"zenthril_device_revocations_total 1",
		"zenthril_key_bundle_claims_total 1",
		"zenthril_prekey_depleted_total 1",
	} {
		if !strings.Contains(body, metric) {
			t.Fatalf("prometheus body missing %q:\n%s", metric, body)
		}
	}
}
