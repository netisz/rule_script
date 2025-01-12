package logger

import (
    "encoding/json"
    "fmt"
    "io"
    "log"
    "os"
    "strings"
    "sync"
    "time"
)

// Level 日志级别
type Level int

const (
    // DEBUG 调试级别
    DEBUG Level = iota
    // INFO 信息级别
    INFO
    // WARN 警告级别
    WARN
    // ERROR 错误级别
    ERROR
    // FATAL 致命错误级别
    FATAL
)

// Fields 定义日志字段类型
type Fields map[string]interface{}

var levelNames = map[Level]string{
    DEBUG: "DEBUG",
    INFO:  "INFO",
    WARN:  "WARN",
    ERROR: "ERROR",
    FATAL: "FATAL",
}

// Logger 日志记录器
type Logger struct {
    mu       sync.Mutex
    level    Level
    output   io.Writer
    prefix   string
    logger   *log.Logger
    colored  bool
    fields   Fields
    jsonMode bool // 是否使用JSON格式输出
}

var (
    // 默认日志记录器
    defaultLogger *Logger
    // ANSI 颜色代码
    colors = map[Level]string{
        DEBUG: "\033[36m", // 青色
        INFO:  "\033[32m", // 绿色
        WARN:  "\033[33m", // 黄色
        ERROR: "\033[31m", // 红色
        FATAL: "\033[35m", // 紫色
    }
    reset = "\033[0m"
    // 日志级别反向映射
    nameToLevel map[string]Level
)

func init() {
    defaultLogger = NewLogger(os.Stdout, "", DEBUG)

    // 初始化反向映射
    nameToLevel = make(map[string]Level, len(levelNames))
    for level, name := range levelNames {
        nameToLevel[name] = level
    }
}

// NewLogger 创建新的日志记录器
func NewLogger(out io.Writer, prefix string, level Level) *Logger {
    return &Logger{
        output:   out,
        prefix:   prefix,
        level:    level,
        logger:   log.New(out, prefix, 0),
        colored:  true,
        fields:   make(Fields),
        jsonMode: false,
    }
}

// WithFields 返回带有指定字段的新日志记录器
func (l *Logger) WithFields(fields Fields) *Logger {
    newLogger := &Logger{
        level:    l.level,
        output:   l.output,
        prefix:   l.prefix,
        logger:   l.logger,
        colored:  l.colored,
        fields:   make(Fields),
        jsonMode: l.jsonMode,
    }

    // 复制现有字段
    for k, v := range l.fields {
        newLogger.fields[k] = v
    }
    // 添加新字段
    for k, v := range fields {
        // 过滤敏感信息
        if filtered := filterSensitiveData(k, v); filtered != nil {
            newLogger.fields[k] = filtered
        }
    }

    return newLogger
}

// WithField 添加单个字段
func (l *Logger) WithField(key string, value interface{}) *Logger {
    return l.WithFields(Fields{key: value})
}

// WithError 添加错误信息
func (l *Logger) WithError(err error) *Logger {
    return l.WithField("error", err.Error())
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
  }

  // SetColored 设置是否启用颜色输出
  func (l *Logger) SetColored(colored bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.colored = colored
  }

// SetJSONMode 设置是否使用JSON格式输出
func (l *Logger) SetJSONMode(enabled bool) {
    l.mu.Lock()
    defer l.mu.Unlock()
    l.jsonMode = enabled
}

// log 输出日志
func (l *Logger) log(level Level, format string, v ...interface{}) {
    if level < l.level {
        return
    }

    l.mu.Lock()
    defer l.mu.Unlock()

    now := time.Now()
    msg := fmt.Sprintf(format, v...)

    if l.jsonMode {
        l.writeJSONLog(level, now, msg)
    } else {
        l.writeTextLog(level, now, msg)
    }

    if level == FATAL {
        os.Exit(1)
    }
}

// writeJSONLog 写入JSON格式日志
func (l *Logger) writeJSONLog(level Level, now time.Time, msg string) {
    logEntry := map[string]interface{}{
        "timestamp": now.Format(time.RFC3339Nano),
        "level":     levelNames[level],
        "message":   msg,
    }

    // 添加字段
    for k, v := range l.fields {
        logEntry[k] = v
    }

    jsonData, err := json.Marshal(logEntry)
    if err != nil {
        // 如果JSON编码失败，回退到普通格式
        l.writeTextLog(level, now, msg)
        return
    }

    l.logger.Println(string(jsonData))
}

// writeTextLog 写入文本格式日志
func (l *Logger) writeTextLog(level Level, now time.Time, msg string) {
    levelName := fmt.Sprintf("[%s]", levelNames[level])
    timestamp := now.Format("2006.01.02 15:04:05.000")

    // 构建字段信息
    var fields string
    if len(l.fields) > 0 {
        parts := make([]string, 0, len(l.fields))
        for k, v := range l.fields {
            parts = append(parts, fmt.Sprintf("%s=%v", k, formatValue(v)))
        }
        fields = " " + strings.Join(parts, " ")
    }

    if l.colored {
        color := colors[level]
        l.logger.Printf("%s%-7s%s %s %s %s", color, levelName, reset, timestamp, msg, fields)
    } else {
        l.logger.Printf("%-7s %s %s %s", levelName, timestamp, msg, fields)
    }
}

// formatValue 格式化字段值
func formatValue(v interface{}) string {
    switch val := v.(type) {
    case string:
        return fmt.Sprintf("%q", val)
    case error:
        return fmt.Sprintf("%q", val.Error())
    case fmt.Stringer:
        return fmt.Sprintf("%q", val.String())
    default:
        return fmt.Sprintf("%v", val)
    }
}

// filterSensitiveData 过滤敏感信息
func filterSensitiveData(key string, value interface{}) interface{} {
    sensitiveKeys := map[string]bool{
        "password":     true,
        "token":       true,
        "credit_card": true,
        "secret":      true,
    }

    if sensitiveKeys[strings.ToLower(key)] {
        return "[FILTERED]"
    }
    return value
}

// Debug 输出调试级别日志
func (l *Logger) Debug(format string, v ...interface{}) {
    l.log(DEBUG, format, v...)
}

// Info 输出信息级别日志
func (l *Logger) Info(format string, v ...interface{}) {
    l.log(INFO, format, v...)
}

// Warn 输出警告级别日志
func (l *Logger) Warn(format string, v ...interface{}) {
    l.log(WARN, format, v...)
}

// Error 输出错误级别日志
func (l *Logger) Error(format string, v ...interface{}) {
    l.log(ERROR, format, v...)
}

// Fatal 输出致命错误级别日志并退出程序
func (l *Logger) Fatal(format string, v ...interface{}) {
    l.log(FATAL, format, v...)
}

// 包级别函数
func Debug(format string, v ...interface{}) {
    defaultLogger.Debug(format, v...)
}

func Info(format string, v ...interface{}) {
    defaultLogger.Info(format, v...)
}

func Warn(format string, v ...interface{}) {
    defaultLogger.Warn(format, v...)
}

func Error(format string, v ...interface{}) {
    defaultLogger.Error(format, v...)
}

func Fatal(format string, v ...interface{}) {
    defaultLogger.Fatal(format, v...)
}

func WithFields(fields Fields) *Logger {
    return defaultLogger.WithFields(fields)
}

func WithField(key string, value interface{}) *Logger {
    return defaultLogger.WithField(key, value)
}

func WithError(err error) *Logger {
    return defaultLogger.WithError(err)
}

// SetLevel 设置日志级别
func SetLevel(level Level) {
    defaultLogger.SetLevel(level)
}

// SetOutput 设置日志输出目标
func SetOutput(w io.Writer) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.output = w
	defaultLogger.logger = log.New(w, defaultLogger.prefix, 0)
  }

// SetColored 设置是否启用颜色输出
func SetColored(colored bool) {
    defaultLogger.SetColored(colored)
}

// SetJSONMode 设置是否使用JSON格式输出
func SetJSONMode(enabled bool) {
    defaultLogger.SetJSONMode(enabled)
}

// ParseLevel 解析日志级别字符串
func ParseLevel(level string) (Level, error) {
    if lvl, ok := nameToLevel[strings.ToUpper(level)]; ok {
        return lvl, nil
    }
    return DEBUG, fmt.Errorf("未知的日志级别: %s", level)
}
