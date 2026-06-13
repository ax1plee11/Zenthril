package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshot := Global().Snapshot()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(snapshot); err != nil {
			http.Error(w, `{"error":"failed to encode metrics"}`, http.StatusInternalServerError)
			return
		}
	}
}

func PrometheusHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snapshot := Global().Snapshot()

		w.Header().Set("Content-Type", "text/plain; version=0.0.4")

		w.Write([]byte("# HELP zenthril_active_connections Current number of active WebSocket connections\n"))
		w.Write([]byte("# TYPE zenthril_active_connections gauge\n"))
		w.Write([]byte("zenthril_active_connections " + formatInt64(snapshot.ActiveConnections) + "\n\n"))

		w.Write([]byte("# HELP zenthril_total_connections Total number of WebSocket connections\n"))
		w.Write([]byte("# TYPE zenthril_total_connections counter\n"))
		w.Write([]byte("zenthril_total_connections " + formatInt64(snapshot.TotalConnections) + "\n\n"))

		w.Write([]byte("# HELP zenthril_total_messages Total number of messages\n"))
		w.Write([]byte("# TYPE zenthril_total_messages counter\n"))
		w.Write([]byte("zenthril_total_messages " + formatInt64(snapshot.TotalMessages) + "\n\n"))

		w.Write([]byte("# HELP zenthril_encryption_ops Total encryption operations\n"))
		w.Write([]byte("# TYPE zenthril_encryption_ops counter\n"))
		w.Write([]byte("zenthril_encryption_ops " + formatInt64(snapshot.EncryptionOps) + "\n\n"))

		w.Write([]byte("# HELP zenthril_avg_encryption_ms Average encryption time in milliseconds\n"))
		w.Write([]byte("# TYPE zenthril_avg_encryption_ms gauge\n"))
		w.Write([]byte("zenthril_avg_encryption_ms " + formatFloat64(snapshot.AvgEncryptionMs) + "\n\n"))

		w.Write([]byte("# HELP zenthril_db_queries Total database queries\n"))
		w.Write([]byte("# TYPE zenthril_db_queries counter\n"))
		w.Write([]byte("zenthril_db_queries " + formatInt64(snapshot.DBQueries) + "\n\n"))

		w.Write([]byte("# HELP zenthril_avg_db_query_ms Average database query time in milliseconds\n"))
		w.Write([]byte("# TYPE zenthril_avg_db_query_ms gauge\n"))
		w.Write([]byte("zenthril_avg_db_query_ms " + formatFloat64(snapshot.AvgDBQueryMs) + "\n\n"))

		w.Write([]byte("# HELP zenthril_db_errors Total database errors\n"))
		w.Write([]byte("# TYPE zenthril_db_errors counter\n"))
		w.Write([]byte("zenthril_db_errors " + formatInt64(snapshot.DBErrors) + "\n\n"))

		w.Write([]byte("# HELP zenthril_http_requests Total HTTP requests\n"))
		w.Write([]byte("# TYPE zenthril_http_requests counter\n"))
		w.Write([]byte("zenthril_http_requests " + formatInt64(snapshot.HTTPRequests) + "\n\n"))

		w.Write([]byte("# HELP zenthril_http_errors Total HTTP errors (4xx, 5xx)\n"))
		w.Write([]byte("# TYPE zenthril_http_errors counter\n"))
		w.Write([]byte("zenthril_http_errors " + formatInt64(snapshot.HTTPErrors) + "\n\n"))

		w.Write([]byte("# HELP zenthril_message_latency_p50_ms Message latency P50 in milliseconds\n"))
		w.Write([]byte("# TYPE zenthril_message_latency_p50_ms gauge\n"))
		w.Write([]byte("zenthril_message_latency_p50_ms " + formatFloat64(snapshot.MessageLatencyP50) + "\n\n"))

		w.Write([]byte("# HELP zenthril_message_latency_p95_ms Message latency P95 in milliseconds\n"))
		w.Write([]byte("# TYPE zenthril_message_latency_p95_ms gauge\n"))
		w.Write([]byte("zenthril_message_latency_p95_ms " + formatFloat64(snapshot.MessageLatencyP95) + "\n\n"))

		w.Write([]byte("# HELP zenthril_message_latency_p99_ms Message latency P99 in milliseconds\n"))
		w.Write([]byte("# TYPE zenthril_message_latency_p99_ms gauge\n"))
		w.Write([]byte("zenthril_message_latency_p99_ms " + formatFloat64(snapshot.MessageLatencyP99) + "\n\n"))

		w.Write([]byte("# HELP zenthril_ws_rejected_total Total rejected WebSocket connections\n"))
		w.Write([]byte("# TYPE zenthril_ws_rejected_total counter\n"))
		w.Write([]byte("zenthril_ws_rejected_total " + formatInt64(snapshot.WSRejected) + "\n\n"))

		w.Write([]byte("# HELP zenthril_ws_rate_limit_hits_total Total WebSocket rate-limit hits\n"))
		w.Write([]byte("# TYPE zenthril_ws_rate_limit_hits_total counter\n"))
		w.Write([]byte("zenthril_ws_rate_limit_hits_total " + formatInt64(snapshot.WSRateLimitHits) + "\n\n"))

		w.Write([]byte("# HELP zenthril_ws_malformed_total Total malformed WebSocket messages\n"))
		w.Write([]byte("# TYPE zenthril_ws_malformed_total counter\n"))
		w.Write([]byte("zenthril_ws_malformed_total " + formatInt64(snapshot.WSMalformed) + "\n\n"))

		w.Write([]byte("# HELP zenthril_ws_forbidden_total Total forbidden WebSocket channel or voice actions\n"))
		w.Write([]byte("# TYPE zenthril_ws_forbidden_total counter\n"))
		w.Write([]byte("zenthril_ws_forbidden_total " + formatInt64(snapshot.WSForbidden) + "\n\n"))

		w.Write([]byte("# HELP zenthril_ws_malformed_closed_total Total WebSocket connections closed after malformed-message threshold\n"))
		w.Write([]byte("# TYPE zenthril_ws_malformed_closed_total counter\n"))
		w.Write([]byte("zenthril_ws_malformed_closed_total " + formatInt64(snapshot.WSMalformedClosed) + "\n\n"))

		w.Write([]byte("# HELP zenthril_readiness_failures_total Total readiness check failures\n"))
		w.Write([]byte("# TYPE zenthril_readiness_failures_total counter\n"))
		w.Write([]byte("zenthril_readiness_failures_total " + formatInt64(snapshot.ReadinessFailures) + "\n\n"))

		w.Write([]byte("# HELP zenthril_device_registrations_total Total E2EE device registrations\n"))
		w.Write([]byte("# TYPE zenthril_device_registrations_total counter\n"))
		w.Write([]byte("zenthril_device_registrations_total " + formatInt64(snapshot.DeviceRegistrations) + "\n\n"))

		w.Write([]byte("# HELP zenthril_device_revocations_total Total E2EE device revocations\n"))
		w.Write([]byte("# TYPE zenthril_device_revocations_total counter\n"))
		w.Write([]byte("zenthril_device_revocations_total " + formatInt64(snapshot.DeviceRevocations) + "\n\n"))

		w.Write([]byte("# HELP zenthril_key_bundle_claims_total Total E2EE key bundle claims\n"))
		w.Write([]byte("# TYPE zenthril_key_bundle_claims_total counter\n"))
		w.Write([]byte("zenthril_key_bundle_claims_total " + formatInt64(snapshot.KeyBundleClaims) + "\n\n"))

		w.Write([]byte("# HELP zenthril_prekey_depleted_total Total key-bundle claims where no one-time prekey was available\n"))
		w.Write([]byte("# TYPE zenthril_prekey_depleted_total counter\n"))
		w.Write([]byte("zenthril_prekey_depleted_total " + formatInt64(snapshot.PreKeyDepleted) + "\n\n"))
	}
}

func formatInt64(v int64) string {
	return fmt.Sprintf("%d", v)
}

func formatFloat64(v float64) string {
	return fmt.Sprintf("%.4f", v)
}
