package main

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// sugar := zap.NewExample().Sugar()
	// sugar.Desugar().Sugar().Infof("Hello my name: %s, age: %d ", "Gia Huy", 23)

	// logger := zap.NewExample()

	// logger.Info("Hello", zap.String("name", "TipsGo"), zap.Int("age", 40))

	// Example
	// logger := zap.NewExample()
	// logger.Info("Hello NewExample")

	// Development
	// logger, _ = zap.NewDevelopment()
	// logger.Info("Hello NewDevelopment")

	// Production
	// logger, _ = zap.NewProduction()
	// logger.Info("Hello NewProduction")

	encoder := getEncoderLog()
	sync := getWriterSync()
	core := zapcore.NewCore(encoder, sync, zapcore.InfoLevel)

	logger := zap.New(core, zap.AddCaller())

	logger.Info("Info log", zap.Int("line", 1))
	logger.Error("Error log", zap.Int("line", 2))
}

// format log
func getEncoderLog() zapcore.Encoder {
	endcodeConfig := zap.NewProductionEncoderConfig()
	endcodeConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	endcodeConfig.TimeKey = "Time"

	endcodeConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	endcodeConfig.EncodeCaller = zapcore.ShortCallerEncoder

	return zapcore.NewJSONEncoder(endcodeConfig)
}

func getWriterSync() zapcore.WriteSyncer {
	file, _ := os.OpenFile("./log/log.txt", os.O_CREATE|os.O_WRONLY, os.ModePerm)
	syncFile := zapcore.AddSync(file)
	syncConsole := zapcore.AddSync(os.Stderr)

	return zapcore.NewMultiWriteSyncer(syncConsole, syncFile)
}
