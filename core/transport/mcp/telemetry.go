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

// Package mcp provides OpenTelemetry telemetry utilities for the MCP protocol.
//
// This file implements telemetry following the MCP Semantic Conventions:
// https://opentelemetry.io/docs/specs/semconv/gen-ai/mcp
//
// To enable telemetry, configure OpenTelemetry in your application before
// creating a ToolboxClient, then pass WithTelemetry(true) as a client option.
// The SDK reads the globally configured TracerProvider and MeterProvider.
// If no provider is configured, OpenTelemetry's no-op implementation is used:
// no data is exported and there is zero overhead.
package mcp

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// MCP semantic convention attribute names.
// See: https://opentelemetry.io/docs/specs/semconv/gen-ai/mcp
const (
	attrMCPMethodName       = "mcp.method.name"
	attrMCPProtocolVersion  = "mcp.protocol.version"
	attrMCPSessionID        = "mcp.session.id"
	attrErrorType           = "error.type"
	attrGenAIToolName       = "gen_ai.tool.name"
	attrGenAIOperationName  = "gen_ai.operation.name"
	attrGenAIPromptName     = "gen_ai.prompt.name"
	attrServerAddress       = "server.address"
	attrServerPort          = "server.port"
	attrNetworkTransport    = "network.transport"
	attrNetworkProtocolName = "network.protocol.name"

	metricClientOperationDuration = "mcp.client.operation.duration"
	metricClientSessionDuration   = "mcp.client.session.duration"
	instrumentationScope          = "toolbox.mcp.sdk"
)

// SpanRef is an alias for trace.Span. Version-specific transport packages
// use this type so they do not need to import the OTel trace package directly.
type SpanRef = trace.Span

// mcpDurationBuckets are the advisory histogram bucket boundaries (seconds).
// See: https://opentelemetry.io/docs/specs/semconv/gen-ai/mcp/#metrics
var mcpDurationBuckets = []float64{0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1, 2, 5, 10, 30, 60, 120, 300}

// InitTelemetry creates a Tracer, operation-duration Histogram, and
// session-duration Histogram from the globally configured OTel providers.
// If no provider is configured, OTel returns no-op instruments (zero
// overhead, no data exported).
func InitTelemetry(sdkVersion string) (trace.Tracer, metric.Float64Histogram, metric.Float64Histogram) {
	tracer := otel.GetTracerProvider().Tracer(
		instrumentationScope,
		trace.WithInstrumentationVersion(sdkVersion),
	)
	meter := otel.GetMeterProvider().Meter(
		instrumentationScope,
		metric.WithInstrumentationVersion(sdkVersion),
	)
	opHist, err := meter.Float64Histogram(
		metricClientOperationDuration,
		metric.WithUnit("s"),
		metric.WithDescription("Duration of MCP client operations (requests/notifications) from the time it was sent until the response or ack is received."),
		metric.WithExplicitBucketBoundaries(mcpDurationBuckets...),
	)
	if err != nil {
		log.Printf("toolbox-sdk: warning: failed to create %s histogram: %v", metricClientOperationDuration, err)
	}
	sessionHist, err := meter.Float64Histogram(
		metricClientSessionDuration,
		metric.WithUnit("s"),
		metric.WithDescription("Total duration of MCP client sessions"),
		metric.WithExplicitBucketBoundaries(mcpDurationBuckets...),
	)
	if err != nil {
		log.Printf("toolbox-sdk: warning: failed to create %s histogram: %v", metricClientSessionDuration, err)
	}
	return tracer, opHist, sessionHist
}

// StartSpan creates an OTel client span for an MCP operation and extracts W3C
// trace context headers for propagation to the server via the JSON-RPC _meta
// field.
//
// The span must be closed by calling EndSpan when the operation completes.
// Returns (nil, "", "") if the tracer is nil or span creation fails.
//
// See: https://opentelemetry.io/docs/specs/semconv/gen-ai/mcp/#context-propagation
func StartSpan(
	ctx context.Context,
	tracer trace.Tracer,
	methodName, protocolVersion, serverURL, toolName string,
) (span trace.Span, traceparent, tracestate string) {
	if tracer == nil {
		return nil, "", ""
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("toolbox-sdk: warning: StartSpan failed: %v", r)
			// clean up span if it was created to prevent memory leaks
			if span != nil {
				span.End()
			}
			span = nil
			traceparent = ""
			tracestate = ""
		}
	}()

	spanName := methodName
	if toolName != "" {
		spanName = methodName + " " + toolName
	}

	_, span = tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindClient))

	host, port, scheme := extractServerInfo(serverURL)
	span.SetAttributes(
		attribute.String(attrMCPMethodName, methodName),
		attribute.String(attrMCPProtocolVersion, protocolVersion),
		attribute.String(attrServerAddress, host),
		attribute.String(attrNetworkProtocolName, scheme),
		attribute.String(attrNetworkTransport, "tcp"),
	)
	if port > 0 {
		span.SetAttributes(attribute.Int(attrServerPort, port))
	}
	if toolName != "" {
		span.SetAttributes(attribute.String(attrGenAIToolName, toolName))
	}
	if methodName == "tools/call" {
		span.SetAttributes(attribute.String(attrGenAIOperationName, "execute_tool"))
	}

	// Activate the span temporarily to extract W3C traceparent/tracestate.
	// These are injected into the JSON-RPC _meta field for server-side linking.
	// See: https://opentelemetry.io/docs/specs/semconv/gen-ai/mcp/#context-propagation
	spanCtx := trace.ContextWithSpan(ctx, span)
	carrier := propagation.MapCarrier{}
	propagation.TraceContext{}.Inject(spanCtx, carrier)
	traceparent = carrier.Get("traceparent")
	tracestate = carrier.Get("tracestate")

	return span, traceparent, tracestate
}

// EndSpan ends a telemetry span. Safe to call with a nil span.
// Sets error status and error.type attribute if err is non-nil.
func EndSpan(span trace.Span, err error) {
	if span == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("toolbox-sdk: warning: EndSpan failed: %v", r)
		}
	}()
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.SetAttributes(attribute.String(attrErrorType, fmt.Sprintf("%T", err)))
	}
	span.End()
}

// RecordOperationDuration records the mcp.client.operation.duration metric.
// Safe to call with a nil histogram.
func RecordOperationDuration(
	ctx context.Context,
	hist metric.Float64Histogram,
	durationSeconds float64,
	methodName, protocolVersion, serverURL, toolName string,
	err error,
) {
	if hist == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("toolbox-sdk: warning: RecordOperationDuration failed: %v", r)
		}
	}()

	host, port, scheme := extractServerInfo(serverURL)
	attrs := []attribute.KeyValue{
		attribute.String(attrMCPMethodName, methodName),
		attribute.String(attrMCPProtocolVersion, protocolVersion),
		attribute.String(attrServerAddress, host),
		attribute.String(attrNetworkProtocolName, scheme),
		attribute.String(attrNetworkTransport, "tcp"),
	}
	if port > 0 {
		attrs = append(attrs, attribute.Int(attrServerPort, port))
	}
	if toolName != "" {
		attrs = append(attrs, attribute.String(attrGenAIToolName, toolName))
	}
	if methodName == "tools/call" {
		attrs = append(attrs, attribute.String(attrGenAIOperationName, "execute_tool"))
	}
	if err != nil {
		attrs = append(attrs, attribute.String(attrErrorType, fmt.Sprintf("%T", err)))
	}
	hist.Record(ctx, durationSeconds, metric.WithAttributes(attrs...))
}

// RecordSessionDuration records the mcp.client.session.duration metric.
// Should be called when the transport session is closed.
// Safe to call with a nil histogram.
func RecordSessionDuration(
	ctx context.Context,
	hist metric.Float64Histogram,
	durationSeconds float64,
	protocolVersion, serverURL string,
	err error,
) {
	if hist == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("toolbox-sdk: warning: RecordSessionDuration failed: %v", r)
		}
	}()

	host, port, scheme := extractServerInfo(serverURL)
	attrs := []attribute.KeyValue{
		attribute.String(attrMCPProtocolVersion, protocolVersion),
		attribute.String(attrServerAddress, host),
		attribute.String(attrNetworkProtocolName, scheme),
		attribute.String(attrNetworkTransport, "tcp"),
	}
	if port > 0 {
		attrs = append(attrs, attribute.Int(attrServerPort, port))
	}
	if err != nil {
		attrs = append(attrs, attribute.String(attrErrorType, fmt.Sprintf("%T", err)))
	}
	hist.Record(ctx, durationSeconds, metric.WithAttributes(attrs...))
}

// RecordErrorFromJSONRPC marks the span as failed using a JSON-RPC error
// response. Sets error.type to "jsonrpc.error.{code}" on the span.
func RecordErrorFromJSONRPC(span trace.Span, errorCode int, errorMessage string) {
	span.SetStatus(codes.Error, errorMessage)
	span.SetAttributes(attribute.String(attrErrorType, fmt.Sprintf("jsonrpc.error.%d", errorCode)))
}

// extractServerInfo parses rawURL and returns the hostname, port, and scheme.
func extractServerInfo(rawURL string) (host string, port int, scheme string) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL, 0, "http"
	}
	scheme = u.Scheme
	if scheme == "" {
		scheme = "http"
	}
	host = u.Hostname()
	if host == "" {
		host = u.Host
	}
	portStr := u.Port()
	if portStr != "" {
		fmt.Sscanf(portStr, "%d", &port) //nolint:errcheck
	}
	return
}
