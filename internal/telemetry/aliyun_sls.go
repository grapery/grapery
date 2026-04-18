package telemetry

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	sls "github.com/aliyun/aliyun-log-go-sdk"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap/zapcore"
)

// SLSConfig holds the configuration for Alibaba Cloud Log Service
type SLSConfig struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	Project         string
	Logstore        string
	Topic           string
	Source          string
}

// SLSLog represents a single log entry for SLS (internal representation)
type SLSLog struct {
	Time     int64
	Contents map[string]string
}

// SLSCore implements zapcore.Core for sending logs to Alibaba Cloud SLS
type SLSCore struct {
	config     SLSConfig
	level      zapcore.Level
	fields     []zapcore.Field
	client     sls.ClientInterface
	mu         sync.Mutex
	buffer     []SLSLog
	bufferSize int
	flushTick  *time.Ticker
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

// NewSLSCore creates a new SLS core for zap logger
func NewSLSCore(config SLSConfig, level zapcore.Level) (*SLSCore, error) {
	// Get source from hostname if not specified
	if config.Source == "" {
		hostname, err := os.Hostname()
		if err != nil {
			config.Source = "unknown"
		} else {
			config.Source = hostname
		}
	}

	// Create official SLS client
	client := sls.CreateNormalInterface(config.Endpoint, config.AccessKeyID, config.AccessKeySecret, "")

	core := &SLSCore{
		config:     config,
		level:      level,
		fields:     make([]zapcore.Field, 0),
		client:     client,
		buffer:     make([]SLSLog, 0, 100),
		bufferSize: 100,
		stopChan:   make(chan struct{}),
	}

	// Start background flush goroutine
	core.flushTick = time.NewTicker(5 * time.Second)
	core.wg.Add(1)
	go core.backgroundFlush()

	return core, nil
}

// backgroundFlush flushes logs periodically
func (c *SLSCore) backgroundFlush() {
	defer c.wg.Done()
	for {
		select {
		case <-c.flushTick.C:
			c.flush()
		case <-c.stopChan:
			c.flush() // Final flush
			return
		}
	}
}

// Enabled returns true if the given level is at or above the core's level
func (c *SLSCore) Enabled(level zapcore.Level) bool {
	return level >= c.level
}

// With adds structured context to the Core
func (c *SLSCore) With(fields []zapcore.Field) zapcore.Core {
	clone := &SLSCore{
		config:     c.config,
		level:      c.level,
		client:     c.client,
		buffer:     c.buffer,
		bufferSize: c.bufferSize,
		flushTick:  c.flushTick,
		stopChan:   c.stopChan,
		fields:     make([]zapcore.Field, len(c.fields)+len(fields)),
	}
	copy(clone.fields, c.fields)
	copy(clone.fields[len(c.fields):], fields)
	return clone
}

// Check determines whether the supplied Entry should be logged
func (c *SLSCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}
	return ce
}

// Write serializes the Entry and any Fields to SLS
func (c *SLSCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// Combine core fields with entry fields
	allFields := make([]zapcore.Field, 0, len(c.fields)+len(fields))
	allFields = append(allFields, c.fields...)
	allFields = append(allFields, fields...)

	// Build log content
	contents := make(map[string]string)
	contents["level"] = entry.Level.String()
	contents["time"] = entry.Time.Format(time.RFC3339Nano)
	contents["logger"] = entry.LoggerName
	contents["message"] = entry.Message
	contents["caller"] = entry.Caller.String()

	if entry.Stack != "" {
		contents["stack"] = entry.Stack
	}

	// Add custom fields
	for _, field := range allFields {
		contents[field.Key] = fieldToString(field)
	}

	log := SLSLog{
		Time:     entry.Time.Unix(),
		Contents: contents,
	}

	c.mu.Lock()
	c.buffer = append(c.buffer, log)
	shouldFlush := len(c.buffer) >= c.bufferSize
	c.mu.Unlock()

	if shouldFlush {
		go c.flush()
	}

	return nil
}

// flush sends buffered logs to SLS using official SDK
func (c *SLSCore) flush() {
	c.mu.Lock()
	if len(c.buffer) == 0 {
		c.mu.Unlock()
		return
	}
	logs := c.buffer
	c.buffer = make([]SLSLog, 0, c.bufferSize)
	c.mu.Unlock()

	// Convert internal logs to official SDK format
	slsLogs := make([]*sls.Log, 0, len(logs))
	for _, log := range logs {
		contents := make([]*sls.LogContent, 0, len(log.Contents))
		for k, v := range log.Contents {
			contents = append(contents, &sls.LogContent{
				Key:   proto.String(k),
				Value: proto.String(v), // Official SDK ensures Value is always string
			})
		}

		slsLogs = append(slsLogs, &sls.Log{
			Time:     proto.Uint32(uint32(log.Time)),
			Contents: contents,
		})
	}

	// Create log group using official SDK format
	logGroup := &sls.LogGroup{
		Logs:   slsLogs,
		Topic:  proto.String(c.config.Topic),
		Source: proto.String(c.config.Source),
	}

	// Send logs using official SDK
	err := c.client.PutLogs(c.config.Project, c.config.Logstore, logGroup)
	if err != nil {
		fmt.Printf("sendToSLS error: %v\n", err)
	} else {
		fmt.Println("sendToSLS success")
	}
}

// Sync flushes buffered logs
func (c *SLSCore) Sync() error {
	c.flush()
	return nil
}

// Close closes the SLS core
func (c *SLSCore) Close() {
	if c.flushTick != nil {
		c.flushTick.Stop()
	}
	close(c.stopChan)
	c.wg.Wait()
}

// fieldToString converts a zap field to string representation
// Always returns a string to ensure SLS API compatibility
func fieldToString(field zapcore.Field) string {
	// Try Interface first for complex types
	if field.Interface != nil {
		switch v := field.Interface.(type) {
		case float64:
			return fmt.Sprintf("%g", v)
		case float32:
			return fmt.Sprintf("%g", v)
		case time.Time:
			return v.Format(time.RFC3339Nano)
		case error:
			return v.Error()
		case string:
			return v
		default:
			// For complex types, try JSON marshaling
			if data, err := json.Marshal(v); err == nil {
				return string(data)
			}
			return fmt.Sprintf("%v", v)
		}
	}

	// Handle primitive types based on field type
	switch field.Type {
	case zapcore.StringType:
		return field.String
	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
		return fmt.Sprintf("%d", field.Integer)
	case zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type:
		return fmt.Sprintf("%d", field.Integer)
	case zapcore.Float64Type:
		// Float64 is stored as bits in Integer field
		// Use math.Float64frombits to extract the value
		return fmt.Sprintf("%g", math.Float64frombits(uint64(field.Integer)))
	case zapcore.Float32Type:
		// Float32 is stored as bits in Integer field
		return fmt.Sprintf("%g", math.Float32frombits(uint32(field.Integer)))
	case zapcore.BoolType:
		if field.Integer == 1 {
			return "true"
		}
		return "false"
	case zapcore.TimeType:
		return time.Unix(0, field.Integer).Format(time.RFC3339Nano)
	case zapcore.DurationType:
		return time.Duration(field.Integer).String()
	case zapcore.ErrorType:
		return ""
	case zapcore.ReflectType:
		return fmt.Sprintf("%v", field.Interface)
	default:
		if field.String != "" {
			return field.String
		}
		return fmt.Sprintf("%v", field.Interface)
	}
}

// SLSWriter implements io.Writer for simple log writing to SLS
type SLSWriter struct {
	core *SLSCore
}

// NewSLSWriter creates a new SLS writer
func NewSLSWriter(config SLSConfig) (*SLSWriter, error) {
	core, err := NewSLSCore(config, zapcore.InfoLevel)
	if err != nil {
		return nil, err
	}
	return &SLSWriter{core: core}, nil
}

// Write implements io.Writer
func (w *SLSWriter) Write(p []byte) (n int, err error) {
	entry := zapcore.Entry{
		Time:    time.Now(),
		Level:   zapcore.InfoLevel,
		Message: string(p),
	}
	if err := w.core.Write(entry, nil); err != nil {
		return 0, err
	}
	return len(p), nil
}

// Close closes the SLS writer
func (w *SLSWriter) Close() {
	w.core.Close()
}
