package main

import "testing"

func TestServerMetricsRecordRequest(t *testing.T) {
	metrics := &serverMetrics{}

	metrics.recordRequest(true)
	metrics.recordRequest(false)

	if got := metrics.requestCount; got != 2 {
		t.Fatalf("expected 2 requests, got %d", got)
	}
	if got := metrics.errorCount; got != 1 {
		t.Fatalf("expected 1 error, got %d", got)
	}
}
