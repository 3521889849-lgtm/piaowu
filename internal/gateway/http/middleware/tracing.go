package middleware

import (
	"context"
	"net/http"
	"strings"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	"github.com/cloudwego/hertz/pkg/app"
)

func Tracing() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		tracer := opentracing.GlobalTracer()

		carrier := opentracing.HTTPHeadersCarrier(http.Header{})
		c.Request.Header.VisitAll(func(k, v []byte) {
			carrier.Set(string(k), string(v))
		})

		var span opentracing.Span
		spanCtx, err := tracer.Extract(opentracing.HTTPHeaders, carrier)
		operationName := strings.ToUpper(string(c.Request.Method())) + " " + string(c.Request.URI().Path())
		if err == nil {
			span = tracer.StartSpan(operationName, ext.RPCServerOption(spanCtx))
		} else {
			span = tracer.StartSpan(operationName)
		}

		ext.SpanKindRPCServer.Set(span)
		ext.HTTPMethod.Set(span, strings.ToUpper(string(c.Request.Method())))
		ext.HTTPUrl.Set(span, string(c.Request.URI().Path()))

		ctxWithSpan := opentracing.ContextWithSpan(ctx, span)
		c.Next(ctxWithSpan)

		statusCode := c.Response.StatusCode()
		ext.HTTPStatusCode.Set(span, uint16(statusCode))
		if statusCode >= 500 {
			ext.Error.Set(span, true)
		}
		span.Finish()
	}
}

