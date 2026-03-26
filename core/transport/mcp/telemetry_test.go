// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build unit

package mcp

import (
	"context"
	"errors"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// newTestTracer returns a real SDK tracer backed by an in-memory span exporter
// so tests can assert on span attributes without a real OTel backend.
func newTestTracer(t *testing.T) (*tracetest.InMemoryExporter, *sdktrace.TracerProvider) {
	t.Helper()
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	return exp, tp
}

// --------------------------------------------------------------------------
// extractServerInfo
// --------------------------------------------------------------------------

func TestExtractServerInfo(t *testing.T) {
	tests := []struct {
		name       string
		rawURL     string
		wantHost   string
		wantPort   int
		wantScheme string
	}{
		{
			name:       "http with explicit port",
			rawURL:     "http://localhost:8080/mcp/",
			wantHost:   "localhost",
			wantPort:   8080,
			wantScheme: "http",
		},
		{
			name:       "https without explicit port",
			rawURL:     "https://example.com/mcp/",
			wantHost:   "example.com",
			wantPort:   0,
			wantScheme: "https",
		},
		{
			name:       "path-only URL defaults scheme to http",
			rawURL:     "/mcp/",
			wantHost:   "",
			wantPort:   0,
			wantScheme: "http",
		},
		{
			name:       "empty string returns defaults",
			rawURL:     "",
			wantHost:   "",
			wantPort:   0,
			wantScheme: "http",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			host, port, scheme := extractServerInfo(tc.rawURL)
			if host != tc.wantHost {
				t.Errorf("host: got %q, want %q", host, tc.wantHost)
			}
			if port != tc.wantPort {
				t.Errorf("port: got %d, want %d", port, tc.wantPort)
			}
			if scheme != tc.wantScheme {
				t.Errorf("scheme: got %q, want %q", scheme, tc.wantScheme)
			}
		})
	}
}

// --------------------------------------------------------------------------
// InitTelemetry
// --------------------------------------------------------------------------

func TestInitTelemetry_ReturnsInstruments(t *testing.T) {
	tracer, opHist, sessHist := InitTelemetry("v0.1.0")
	if tracer == nil {
		t.Error("expected non-nil tracer")
	}
	if opHist == nil {
		t.Error("expected non-nil operation duration histogram")
	}
	if sessHist == nil {
		t.Error("expected non-nil session duration histogram")
	}
}

// --------------------------------------------------------------------------
// StartSpan
// --------------------------------------------------------------------------

func TestStartSpan_SpanAttributes(t *testing.T) {
	exp, tp := newTestTracer(t)
	tracer := tp.Tracer(instrumentationScope)

	ctx := context.Background()
	span, _, _ := StartSpan(ctx, tracer, "tools/call", "2024-11-05", "http://localhost:8080/mcp/", "my-tool")
	EndSpan(span, nil)

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	got := spans[0]

	if got.Name != "tools/call my-tool" {
		t.Errorf("span name: got %q, want %q", got.Name, "tools/call my-tool")
	}

	attrMap := make(map[string]string)
	for _, a := range got.Attributes {
		attrMap[string(a.Key)] = a.Value.AsString()
	}

	checks := map[string]string{
		attrMCPMethodName:       "tools/call",
		attrMCPProtocolVersion:  "2024-11-05",
		attrGenAIToolName:       "my-tool",
		attrGenAIOperationName:  "execute_tool",
		attrServerAddress:       "localhost",
		attrNetworkProtocolName: "http",
		attrNetworkTransport:    "tcp",
	}
	for key, want := range checks {
		if got := attrMap[key]; got != want {
			t.Errorf("attribute %s: got %q, want %q", key, got, want)
		}
	}
}

func TestStartSpan_NoToolName(t *testing.T) {
	exp, tp := newTestTracer(t)
	tracer := tp.Tracer(instrumentationScope)

	ctx := context.Background()
	span, _, _ := StartSpan(ctx, tracer, "tools/list", "2024-11-05", "http://localhost:8080/mcp/", "")
	EndSpan(span, nil)

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Name != "tools/list" {
		t.Errorf("span name: got %q, want %q", spans[0].Name, "tools/list")
	}
}

func TestStartSpan_NilTracer_ReturnsNilAndEmptyStrings(t *testing.T) {
	ctx := context.Background()
	span, traceparent, tracestate := StartSpan(ctx, nil, "tools/call", "2024-11-05", "http://localhost/", "tool")

	if span != nil {
		t.Error("expected nil span for nil tracer")
	}
	if traceparent != "" {
		t.Error("expected empty traceparent for nil tracer")
	}
	if tracestate != "" {
		t.Error("expected empty tracestate for nil tracer")
	}
}

func TestStartSpan_PropagatesTraceContext(t *testing.T) {
	_, tp := newTestTracer(t)
	tracer := tp.Tracer(instrumentationScope)

	ctx := context.Background()
	span, traceparent, _ := StartSpan(ctx, tracer, "tools/call", "2024-11-05", "http://localhost:8080/mcp/", "my-tool")
	EndSpan(span, nil)

	// traceparent should be a non-empty W3C header (format: 00-<trace>-<span>-<flags>)
	if traceparent == "" {
		t.Error("expected non-empty traceparent for active span")
	}
}

// --------------------------------------------------------------------------
// EndSpan
// --------------------------------------------------------------------------

func TestEndSpan_NilSpan_DoesNotPanic(t *testing.T) {
	EndSpan(nil, nil)
	EndSpan(nil, errors.New("ignored"))
}

func TestEndSpan_SetsErrorAttributesOnFailure(t *testing.T) {
	exp, tp := newTestTracer(t)
	tracer := tp.Tracer(instrumentationScope)

	ctx := context.Background()
	span, _, _ := StartSpan(ctx, tracer, "tools/call", "2024-11-05", "http://localhost:8080/mcp/", "bad-tool")
	EndSpan(span, errors.New("something went wrong"))

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	found := false
	for _, a := range spans[0].Attributes {
		if string(a.Key) == attrErrorType {
			found = true
		}
	}
	if !found {
		t.Error("expected error.type attribute to be set on span when error is passed")
	}
}

// --------------------------------------------------------------------------
// RecordOperationDuration
// --------------------------------------------------------------------------

func TestRecordOperationDuration_NilHistogram_DoesNotPanic(t *testing.T) {
	ctx := context.Background()
	RecordOperationDuration(ctx, nil, 0.5, "tools/call", "2024-11-05", "http://localhost/mcp/", "tool", nil)
	RecordOperationDuration(ctx, nil, 0.5, "tools/call", "2024-11-05", "http://localhost/mcp/", "tool", errors.New("err"))
}

func TestRecordOperationDuration_RecordsWithAttributes(t *testing.T) {
	ctx := context.Background()
	_, opHist, _ := InitTelemetry("v0.1.0")
	// Smoke test: verify it runs without panicking
	RecordOperationDuration(ctx, opHist, 0.123, "tools/call", "2024-11-05", "http://localhost:8080/mcp/", "my-tool", nil)
	RecordOperationDuration(ctx, opHist, 0.456, "tools/list", "2024-11-05", "http://localhost:8080/mcp/", "", errors.New("err"))
}

// --------------------------------------------------------------------------
// RecordErrorFromJSONRPC
// --------------------------------------------------------------------------

func TestRecordErrorFromJSONRPC(t *testing.T) {
	exp, tp := newTestTracer(t)
	tracer := tp.Tracer(instrumentationScope)

	ctx := context.Background()
	_, span := tracer.Start(ctx, "test-span")
	RecordErrorFromJSONRPC(span, -32600, "Invalid Request")
	span.End()

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	attrMap := make(map[string]string)
	for _, a := range spans[0].Attributes {
		attrMap[string(a.Key)] = a.Value.AsString()
	}

	wantErrorType := "jsonrpc.error.-32600"
	if got := attrMap[attrErrorType]; got != wantErrorType {
		t.Errorf("error.type: got %q, want %q", got, wantErrorType)
	}
}

// --------------------------------------------------------------------------
// RecordSessionDuration
// --------------------------------------------------------------------------

func TestRecordSessionDuration_NilHistogram_DoesNotPanic(t *testing.T) {
	ctx := context.Background()
	RecordSessionDuration(ctx, nil, 1.5, "2024-11-05", "http://localhost/mcp/", nil)
	RecordSessionDuration(ctx, nil, 1.5, "2024-11-05", "http://localhost/mcp/", errors.New("err"))
}

func TestRecordSessionDuration_WithError(t *testing.T) {
	ctx := context.Background()
	_, _, sessHist := InitTelemetry("v0.1.0")
	RecordSessionDuration(ctx, sessHist, 3.14, "2024-11-05", "http://localhost:8080/mcp/", errors.New("connection reset"))
}

func TestRecordSessionDuration_NoError(t *testing.T) {
	ctx := context.Background()
	_, _, sessHist := InitTelemetry("v0.1.0")
	RecordSessionDuration(ctx, sessHist, 10.0, "2025-06-18", "https://example.com/mcp/", nil)
}
