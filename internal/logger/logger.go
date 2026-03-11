package logger

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	red    = "\x1b[31m"
	yellow = "\x1b[33m"
	reset  = "\x1b[0m"
)

func NewZapLogger(service string) (*zap.Logger, func()) {
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:    "time",
		LevelKey:   "level",
		NameKey:    "logger",
		CallerKey:  "caller",
		MessageKey: "msg",
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Local().Format("2006/01/02 15:04:05"))
		},
		EncodeLevel: func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
			switch level {
			case zapcore.ErrorLevel,
				zapcore.DPanicLevel,
				zapcore.PanicLevel,
				zapcore.FatalLevel:
				enc.AppendString(red + level.CapitalString() + reset)
			case zapcore.WarnLevel:
				enc.AppendString(yellow + level.CapitalString() + reset)
			default:
				enc.AppendString(level.CapitalString())
			}
		},
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.Lock(zapcore.AddSync(zapcore.AddSync(os.Stdout))),
		zap.InfoLevel,
	)

	logger := zap.New(core).Named(service)
	cleanup := func() { _ = logger.Sync() }

	return logger, cleanup
}
