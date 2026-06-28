package logger

import (
	"context"

	"go.uber.org/atomic"
	"go.uber.org/zap"
)

const ctxLoggerKey = "_logger"

func FromContext(ctx context.Context) *zap.SugaredLogger {
	log, ok := ctx.Value(ctxLoggerKey).(*zap.SugaredLogger)
	if !ok || log == nil {
		return DefaultLogger.Load()
	}
	return log
}

func ToContext(ctx context.Context, log *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, ctxLoggerKey, log)
}

var DefaultLogger = atomic.NewPointer[zap.SugaredLogger](zap.NewNop().Sugar())

var skipOne = zap.AddCallerSkip(1)

func Debugw(ctx context.Context, msg string, fields ...any) {
	FromContext(ctx).WithOptions(skipOne).Debugw(msg, fields...)
}

func Infow(ctx context.Context, msg string, fields ...any) {
	FromContext(ctx).WithOptions(skipOne).Infow(msg, fields...)
}

func Warnw(ctx context.Context, msg string, fields ...any) {
	FromContext(ctx).WithOptions(skipOne).Warnw(msg, fields...)
}

func Errorw(ctx context.Context, msg string, fields ...any) {
	FromContext(ctx).WithOptions(skipOne).Errorw(msg, fields...)
}
