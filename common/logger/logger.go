package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log = zap.NewNop()

// InitLogger 初始化全局 Zap Logger
// env: 开发环境(dev)会同时输出到控制台(高亮)，生产环境(prod)只输出 JSON 到文件
func InitLogger(env string, logPath string) {
	// 配置日志轮转机制
	hook := lumberjack.Logger{
		Filename:   logPath, // 日志文件路径
		MaxSize:    100,     // 每个日志文件最大尺寸（MB）
		MaxBackups: 30,      // 保留旧文件最大个数
		MaxAge:     7,       // 保留旧文件最大天数
		Compress:   true,    // 是否压缩旧文件 (zip)
	}

	// 编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder, // 小写级别 (info, error)
		EncodeTime:     zapcore.ISO8601TimeEncoder,    // ISO8601 时间格式
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,    // 简短调用路径
	}

	var core zapcore.Core

	if env == "dev" {
		// 开发环境：Console 高亮编码器 + JSON 文件编码器
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // 终端颜色高亮
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		fileEncoder := zapcore.NewJSONEncoder(encoderConfig)

		core = zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zap.DebugLevel),
			zapcore.NewCore(fileEncoder, zapcore.AddSync(&hook), zap.DebugLevel),
		)
	} else {
		// 生产环境：仅 JSON 文件编码器
		fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
		core = zapcore.NewCore(fileEncoder, zapcore.AddSync(&hook), zap.InfoLevel)
	}

	// 添加调用者信息和堆栈跟踪
	Log = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
}
