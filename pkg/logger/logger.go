package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

// Initialize creates a new logger instance with the given configuration
func Initialize(development bool) error {
	// Ensure the logs directory exists
	logDir := filepath.Join(os.ExpandEnv("$HOME"), ".config", "pair", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// Configure the file logger
	logPath := filepath.Join(logDir, "app.log")

	// Remove old log file if it exists
	if err := os.Remove(logPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove old log file: %v", err)
	}

	fileConfig := getFileLoggerConfig(logPath, development)
	fileLogger, err := fileConfig.Build(zap.AddCallerSkip(1))
	if err != nil {
		return fmt.Errorf("failed to create file logger: %v", err)
	}

	log = fileLogger
	return nil
}

func getFileLoggerConfig(logPath string, development bool) zap.Config {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(zapcore.DebugLevel),
		Development:      development,
		Encoding:         "json",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{logPath},
		ErrorOutputPaths: []string{logPath},
	}

	return config
}

// GetLogger returns the global logger instance
func GetLogger() *zap.Logger {
	return log
}

// Debug logs a debug message to file only
func Debug(msg string, fields ...zap.Field) {
	log.Debug(msg, fields...)
}

// Info logs an info message to both file and console
func Info(msg string, fields ...zap.Field) {
	log.Info(msg, fields...)
	printConsoleInfo("", msg, fields...)
}

// Warn logs a warning message to both file and console
func Warn(msg string, fields ...zap.Field) {
	log.Warn(msg, fields...)
	printConsoleWarn(msg, fields...)
}

// Error logs an error message to both file and console
func Error(msg string, fields ...zap.Field) {
	log.Error(msg, fields...)
	printConsoleError(msg, fields...)
}

// Fatal logs a fatal message to both file and console, then exits
func Fatal(msg string, fields ...zap.Field) {
	// Print to console first to ensure it's visible
	printConsoleFatal(msg, fields...)
	// Then log to file and exit
	log.Fatal(msg, fields...)
}

func formatFields(fields []zap.Field) []string {
	var lines []string
	for _, field := range fields {
		var value interface{}
		switch field.Type {
		case zapcore.StringType:
			value = field.String
		case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
			value = field.Integer
		case zapcore.Float64Type, zapcore.Float32Type:
			value = field.Integer
		case zapcore.ErrorType:
			if field.Interface != nil {
				if err, ok := field.Interface.(error); ok {
					value = err.Error()
				}
			}
		default:
			value = field.Interface
		}
		lines = append(lines, fmt.Sprintf("  %s: %v", field.Key, value))
	}
	return lines
}

// printConsoleInfo prints a nicely formatted info message to the console
func printConsoleInfo(symbol, msg string, fields ...zap.Field) {
	fmt.Printf("\x1b[36mInfo:\x1b[0m %s\n", msg)
	if len(fields) > 0 {
		for _, line := range formatFields(fields) {
			fmt.Printf("\x1b[36m%s\x1b[0m\n", line)
		}
	}
}

// printConsoleWarn prints a nicely formatted warning message to the console
func printConsoleWarn(msg string, fields ...zap.Field) {
	fmt.Printf("\x1b[33mWarn:\x1b[0m %s\n", msg)
	if len(fields) > 0 {
		for _, line := range formatFields(fields) {
			fmt.Printf("\x1b[33m%s\x1b[0m\n", line)
		}
	}
}

// printConsoleError prints a nicely formatted error message to the console
func printConsoleError(msg string, fields ...zap.Field) {
	fmt.Printf("\x1b[31m✗ Error:\x1b[0m %s\n", msg)
	if len(fields) > 0 {
		fmt.Println("\x1b[90mDetails:\x1b[0m")
		for _, line := range formatFields(fields) {
			fmt.Printf("\x1b[90m%s\x1b[0m\n", line)
		}
	}
}

// printConsoleFatal prints a nicely formatted fatal message to the console
func printConsoleFatal(msg string, fields ...zap.Field) {
	fmt.Printf("\x1b[31;1m✗ FATAL:\x1b[0m %s\n", msg)
	if len(fields) > 0 {
		fmt.Println("\x1b[90mDetails:\x1b[0m")
		for _, line := range formatFields(fields) {
			fmt.Printf("\x1b[90m%s\x1b[0m\n", line)
		}
	}
}
