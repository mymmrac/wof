package logger

import (
	"context"

	"go.uber.org/atomic"
	"go.uber.org/zap"
)

var loggerKey = loggerKeyType{} //nolint:gochecknoglobals

type loggerKeyType struct{}

func FromContext(ctx context.Context) *zap.SugaredLogger {
	log, ok := ctx.Value(loggerKey).(*zap.SugaredLogger)
	if !ok || log == nil {
		return DefaultLogger.Load()
	}
	return log
}

func ToContext(ctx context.Context, log *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, loggerKey, log)
}

//nolint:gochecknoglobals
var DefaultLogger = atomic.NewPointer[zap.SugaredLogger](zap.NewNop().Sugar())

//nolint:gochecknoglobals
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
