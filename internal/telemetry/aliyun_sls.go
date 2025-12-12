package telemetry

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

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

// SLSLog represents a single log entry for SLS
type SLSLog struct {
	Time     int64             `json:"time"`
	Contents map[string]string `json:"contents"`
}

// SLSLogGroup represents a group of logs for SLS
type SLSLogGroup struct {
	Topic  string   `json:"topic"`
	Source string   `json:"source"`
	Logs   []SLSLog `json:"logs"`
}

// SLSCore implements zapcore.Core for sending logs to Alibaba Cloud SLS
type SLSCore struct {
	config     SLSConfig
	level      zapcore.Level
	fields     []zapcore.Field
	httpClient *http.Client
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

	core := &SLSCore{
		config:     config,
		level:      level,
		fields:     make([]zapcore.Field, 0),
		httpClient: &http.Client{Timeout: 10 * time.Second},
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
		httpClient: c.httpClient,
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

// flush sends buffered logs to SLS
func (c *SLSCore) flush() {
	c.mu.Lock()
	if len(c.buffer) == 0 {
		c.mu.Unlock()
		return
	}
	logs := c.buffer
	c.buffer = make([]SLSLog, 0, c.bufferSize)
	c.mu.Unlock()

	logGroup := SLSLogGroup{
		Topic:  c.config.Topic,
		Source: c.config.Source,
		Logs:   logs,
	}

	_ = c.sendToSLS(logGroup)
}

// sendToSLS sends a log group to SLS via HTTP API
func (c *SLSCore) sendToSLS(logGroup SLSLogGroup) error {
	body, err := json.Marshal(logGroup)
	if err != nil {
		return fmt.Errorf("failed to marshal log group: %w", err)
	}

	url := fmt.Sprintf("https://%s.%s/logstores/%s/shards/lb",
		c.config.Project, c.config.Endpoint, c.config.Logstore)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Calculate MD5 of body
	bodyMD5 := md5.Sum(body)
	contentMD5 := strings.ToUpper(fmt.Sprintf("%x", bodyMD5))

	now := time.Now().UTC().Format(http.TimeFormat)

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-MD5", contentMD5)
	req.Header.Set("Date", now)
	req.Header.Set("x-log-apiversion", "0.6.0")
	req.Header.Set("x-log-bodyrawsize", fmt.Sprintf("%d", len(body)))
	req.Header.Set("Host", fmt.Sprintf("%s.%s", c.config.Project, c.config.Endpoint))

	// Sign the request
	signature := c.signRequest(req, contentMD5)
	req.Header.Set("Authorization", fmt.Sprintf("LOG %s:%s", c.config.AccessKeyID, signature))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("SLS returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// signRequest creates the signature for SLS API
func (c *SLSCore) signRequest(req *http.Request, contentMD5 string) string {
	// Build string to sign
	var headers []string
	for key := range req.Header {
		lowerKey := strings.ToLower(key)
		if strings.HasPrefix(lowerKey, "x-log-") || strings.HasPrefix(lowerKey, "x-acs-") {
			headers = append(headers, lowerKey)
		}
	}
	sort.Strings(headers)

	var canonicalHeaders strings.Builder
	for _, key := range headers {
		canonicalHeaders.WriteString(key)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(req.Header.Get(key))
		canonicalHeaders.WriteString("\n")
	}

	stringToSign := fmt.Sprintf("%s\n%s\n%s\n%s\n%s%s",
		req.Method,
		contentMD5,
		req.Header.Get("Content-Type"),
		req.Header.Get("Date"),
		canonicalHeaders.String(),
		req.URL.Path,
	)

	// HMAC-SHA1
	mac := hmac.New(sha1.New, []byte(c.config.AccessKeySecret))
	mac.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
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
func fieldToString(field zapcore.Field) string {
	switch field.Type {
	case zapcore.StringType:
		return field.String
	case zapcore.Int64Type, zapcore.Int32Type, zapcore.Int16Type, zapcore.Int8Type:
		return fmt.Sprintf("%d", field.Integer)
	case zapcore.Uint64Type, zapcore.Uint32Type, zapcore.Uint16Type, zapcore.Uint8Type:
		return fmt.Sprintf("%d", field.Integer)
	case zapcore.Float64Type:
		return fmt.Sprintf("%f", float64(field.Integer))
	case zapcore.Float32Type:
		return fmt.Sprintf("%f", float32(field.Integer))
	case zapcore.BoolType:
		if field.Integer == 1 {
			return "true"
		}
		return "false"
	case zapcore.TimeType:
		if field.Interface != nil {
			if t, ok := field.Interface.(time.Time); ok {
				return t.Format(time.RFC3339Nano)
			}
		}
		return time.Unix(0, field.Integer).Format(time.RFC3339Nano)
	case zapcore.DurationType:
		return time.Duration(field.Integer).String()
	case zapcore.ErrorType:
		if field.Interface != nil {
			if e, ok := field.Interface.(error); ok {
				return e.Error()
			}
		}
		return ""
	case zapcore.ReflectType:
		if field.Interface != nil {
			if data, err := json.Marshal(field.Interface); err == nil {
				return string(data)
			}
		}
		return fmt.Sprintf("%v", field.Interface)
	default:
		if field.Interface != nil {
			if data, err := json.Marshal(field.Interface); err == nil {
				return string(data)
			}
			return fmt.Sprintf("%v", field.Interface)
		}
		return field.String
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
