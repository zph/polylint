package polylint

import "go.uber.org/zap"

var logz *zap.SugaredLogger

func init() {
	logger, _ := zap.NewProductionConfig().Build(
		zap.WithCaller(true),
	)

	zap.NewAtomicLevelAt(zap.DebugLevel)
	defer logger.Sync() // flushes buffer, if any
	logz = logger.Sugar()
	logz.Debugw("polylint initialized", "version", "0.0.2")
}
