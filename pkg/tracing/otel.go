package tracing

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type otelTracer struct {
	tracer   trace.Tracer
	provider *sdktrace.TracerProvider
	config   *Config
}

func initOTELTracer(cfg *Config) (Tracer, error) {
	exporter, err := newExporter(cfg)
	if err != nil {
		return nil, fmt.Errorf("tracing: create exporter: %w", err)
	}

	if exporter == nil {
		return NewNoOpTracer(), nil
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("tracing: create resource: %w", err)
	}

	sampler := sdktrace.TraceIDRatioBased(cfg.SampleRate)

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sampler)),
	)

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	otel.SetTracerProvider(provider)

	return &otelTracer{
		tracer:   provider.Tracer(cfg.ServiceName),
		provider: provider,
		config:   cfg,
	}, nil
}

func (t *otelTracer) StartHTTPRequest(r *http.Request) (context.Context, HTTPTransaction) {
	ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))

	ctx, span := t.tracer.Start(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path),
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			semconv.HTTPRequestMethodKey.String(r.Method),
			semconv.URLPath(r.URL.Path),
			semconv.UserAgentOriginal(r.UserAgent()),
		),
	)

	return ctx, &otelHTTPTransaction{span: span, ctx: ctx}
}

func (t *otelTracer) StartExternalCall(ctx context.Context, host, method string) (context.Context, func()) {
	ctx, span := t.tracer.Start(ctx, fmt.Sprintf("External %s %s", method, host),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("http.method", method),
			attribute.String("net.peer.name", host),
		),
	)
	return ctx, func() { span.End() }
}

func (t *otelTracer) StartMessage(ctx context.Context, topic, operation string) (context.Context, func()) {
	kind := trace.SpanKindProducer
	if operation == "consume" || operation == "receive" {
		kind = trace.SpanKindConsumer
	}

	ctx, span := t.tracer.Start(ctx, fmt.Sprintf("%s %s", topic, operation),
		trace.WithSpanKind(kind),
		trace.WithAttributes(
			attribute.String("messaging.destination", topic),
			attribute.String("messaging.operation", operation),
		),
	)
	return ctx, func() { span.End() }
}

func (t *otelTracer) StartDatabase(ctx context.Context, operation, table string) (context.Context, func()) {
	ctx, span := t.tracer.Start(ctx, fmt.Sprintf("DB %s %s", operation, table),
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String("db.operation", operation),
			attribute.String("db.sql.table", table),
		),
	)
	return ctx, func() { span.End() }
}

func (t *otelTracer) StartSegment(ctx context.Context, name string) (context.Context, func()) {
	ctx, span := t.tracer.Start(ctx, name, trace.WithSpanKind(trace.SpanKindInternal))
	return ctx, func() { span.End() }
}

// The carrier must implement propagation.TextMapCarrier (e.g. http.Header via HeaderCarrier).
func (t *otelTracer) InjectContext(ctx context.Context, carrier interface{}) {
	if c, ok := carrier.(propagation.TextMapCarrier); ok {
		otel.GetTextMapPropagator().Inject(ctx, c)
	}
}

func (t *otelTracer) ExtractContext(ctx context.Context, carrier interface{}) context.Context {
	if c, ok := carrier.(propagation.TextMapCarrier); ok {
		return otel.GetTextMapPropagator().Extract(ctx, c)
	}
	return ctx
}

// IsEnabled returns true — the OTEL tracer is always active once created.
func (t *otelTracer) IsEnabled() bool { return true }

func (t *otelTracer) ShouldTrace(path string) bool {
	for _, excluded := range t.config.ExcludePaths {
		if strings.HasPrefix(path, excluded) {
			return false
		}
	}
	return true
}

func (t *otelTracer) Shutdown(ctx context.Context) error {
	if t.provider == nil {
		return nil
	}
	return t.provider.Shutdown(ctx)
}

type otelHTTPTransaction struct {
	span trace.Span
	ctx  context.Context
}

func (tx *otelHTTPTransaction) Context() context.Context { return tx.ctx }

func (tx *otelHTTPTransaction) SetName(name string) {
	tx.span.SetName(name)
}

func (tx *otelHTTPTransaction) AddAttribute(key, value string) {
	tx.span.SetAttributes(attribute.String(key, value))
}

func (tx *otelHTTPTransaction) AddError(err error) {
	if err != nil {
		tx.span.RecordError(err)
		tx.span.SetStatus(codes.Error, err.Error())
	}
}

func (tx *otelHTTPTransaction) End() {
	tx.span.End()
}
