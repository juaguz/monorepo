package telemetry

import (
	"github.com/monorepo/go/kit/logger"
	"github.com/sirupsen/logrus"
	"time"
)

type Client interface {
	Incr(name string, tags map[string]interface{})
	Decr(name string, tags map[string]interface{})
	Timing(name string, value time.Duration, tags map[string]interface{})
	Count(name string, value int64, tags map[string]interface{})
}

var log *logrus.Logger

func init() {
	log = logger.GetLogger()
}

type LogTelemetry struct{}

func NewTelemetry() *LogTelemetry {
	return &LogTelemetry{}
}

func appendTags(fields, tags map[string]interface{}) map[string]interface{} {
	if tags == nil {
		return fields
	}

	for key, tag := range tags {
		fields[key] = tag
	}

	return fields
}

func (c *LogTelemetry) Incr(name string, tags map[string]interface{}) {
	fields := map[string]interface{}{
		"metric": true,
		"name":   name,
		"type":   "incr",
	}

	fields = appendTags(fields, tags)

	log.WithFields(fields).Log(logrus.InfoLevel, "+1")
}

func (c *LogTelemetry) Decr(name string, tags map[string]interface{}) {
	fields := map[string]interface{}{
		"metric": true,
		"name":   name,
		"type":   "decr",
	}

	fields = appendTags(fields, tags)

	log.WithFields(fields).Log(logrus.InfoLevel, "-1")
}

func (c *LogTelemetry) Count(name string, value int64, tags map[string]interface{}) {
	fields := map[string]interface{}{
		"metric": true,
		"name":   name,
		"type":   "count",
	}

	fields = appendTags(fields, tags)

	log.WithFields(fields).Log(logrus.InfoLevel, value)
}

func (c *LogTelemetry) Timing(name string, value time.Duration, tags map[string]interface{}) {
	fields := map[string]interface{}{
		"metric": true,
		"name":   name,
		"type":   "timing",
	}

	fields = appendTags(fields, tags)

	log.WithFields(fields).Log(logrus.InfoLevel, value.Milliseconds())
}
