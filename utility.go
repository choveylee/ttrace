/**
 * @Author: lidonglin
 * @Description:
 * @File:  utility.go
 * @Version: 1.0.0
 * @Date: 2023/11/14 09:34
 */

package ttrace

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	"go.opentelemetry.io/otel/trace"
)

func init() {
	rand.NewSource(time.Now().UnixNano())
}

func InjectContext(ctx context.Context) (context.Context, error) {
	traceId := NewTraceId()
	spanId := NewSpanId()

	var desTraceId [16]byte
	var desSpanId [8]byte

	span := trace.SpanFromContext(ctx)

	for i := 0; i < 16; i++ {
		desTraceId[i] = traceId[i]
	}

	for i := 0; i < 8; i++ {
		desSpanId[i] = spanId[i]
	}

	spanContext := span.SpanContext().WithTraceID(desTraceId).WithSpanID(desSpanId)

	ctx = trace.ContextWithSpanContext(ctx, spanContext)

	return ctx, nil
}

func InjectTrace(ctx context.Context, traceId, spanId string) (context.Context, error) {
	srcTraceId, err := hex.DecodeString(traceId)
	if err != nil {
		return nil, err
	}

	if len(srcTraceId) != 16 {
		return nil, fmt.Errorf("trace id illegal")
	}

	srcSpanId, err := hex.DecodeString(spanId)
	if err != nil {
		return nil, err
	}

	if len(srcSpanId) != 8 {
		return nil, fmt.Errorf("span id illegal")
	}

	var desTraceId [16]byte
	var desSpanId [8]byte

	span := trace.SpanFromContext(ctx)

	for i := 0; i < 16; i++ {
		desTraceId[i] = srcTraceId[i]
	}

	for i := 0; i < 8; i++ {
		desSpanId[i] = srcSpanId[i]
	}

	spanContext := span.SpanContext().WithTraceID(desTraceId).WithSpanID(desSpanId)

	ctx = trace.ContextWithSpanContext(ctx, spanContext)

	return ctx, nil
}

func NewTraceId() string {
	bytesInit := []byte("0123456789abcdef")

	data := make([]byte, 0)

	for i := 0; i < 32; i++ {
		data = append(data, bytesInit[rand.Intn(len(bytesInit))])
	}

	return string(data)
}

func NewSpanId() string {
	bytesInit := []byte("0123456789abcdef")

	data := make([]byte, 0)

	for i := 0; i < 16; i++ {
		data = append(data, bytesInit[rand.Intn(len(bytesInit))])
	}

	return string(data)
}
