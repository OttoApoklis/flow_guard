package logger

import (
	"fmt"
	"github.com/OttoApoklis/flow_guard/config"
	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
)

// GlobalLogger 日志实例
var GlobalLogger *logrus.Logger

// InitLogger 初始化日志记录器
func InitLogger(cfg *config.Config) error {
	// 读取日志配置
	logConfig := cfg.FlowGuard.LogConfig

	// 创建日志实例
	GlobalLogger = logrus.New()

	// 设置日志级别
	level, err := logrus.ParseLevel(logConfig.Level)
	if err != nil {
		return fmt.Errorf("invalid log level: %v", err)
	}
	GlobalLogger.SetLevel(level)

	// 设置日志输出文件
	GlobalLogger.SetOutput(&lumberjack.Logger{
		Filename:   logConfig.File,
		MaxSize:    logConfig.MaxSize,    // 单个文件的最大大小 (MB)
		MaxBackups: logConfig.MaxBackups, // 保留日志的数量
		MaxAge:     logConfig.MaxAge,     // 保留天数
	})

	return nil
}
