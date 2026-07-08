package metrics

import (
	"context"
	"sync"

	"go.opentelemetry.io/otel/metric"
)

type Int64CounterHelper struct {
	sync.RWMutex
	counters map[string]Int64Counter
}

func NewInt64CounterHelper() *Int64CounterHelper {
	return &Int64CounterHelper{
		counters: make(map[string]Int64Counter),
	}
}

func (c *Int64CounterHelper) Add(ctx context.Context, counterName string, incr int64, options ...metric.AddOption) {
	c.RLock()
	defer c.RUnlock()
	counter, ok := c.counters[counterName]
	if !ok {
		return
	}
	counter.Add(ctx, incr, options...)
}

func (c *Int64CounterHelper) CreateCounter(meter Meter, counterName string, descriptions ...string) error {
	options := make([]metric.Int64CounterOption, 0)
	if len(descriptions) > 0 && descriptions[0] != "" {
		options = append(options, metric.WithDescription(descriptions[0]))
	}
	counter, err := meter.Int64Counter(counterName, options...)
	if err != nil {
		return err
	}
	c.Lock()
	defer c.Unlock()
	c.counters[counterName] = counter
	return nil
}

func (c *Int64CounterHelper) Close(context.Context) {
	c.Lock()
	defer c.Unlock()
	clear(c.counters)
}

type Int64GaugeHelper struct {
	sync.RWMutex
	gauges map[string]Int64Gauge
}

func NewInt64GaugeHelper() *Int64GaugeHelper {
	return &Int64GaugeHelper{
		gauges: make(map[string]Int64Gauge),
	}
}

func (c *Int64GaugeHelper) Record(ctx context.Context, gaugeName string, value int64, options ...metric.RecordOption) {
	c.RLock()
	defer c.RUnlock()
	gauge, ok := c.gauges[gaugeName]
	if !ok {
		return
	}
	gauge.Record(ctx, value, options...)
}

func (c *Int64GaugeHelper) CreateGauge(meter Meter, gaugeName string, descriptions ...string) error {
	options := make([]metric.Int64GaugeOption, 0)
	if len(descriptions) > 0 && descriptions[0] != "" {
		options = append(options, metric.WithDescription(descriptions[0]))
	}
	gauge, err := meter.Int64Gauge(gaugeName, options...)
	if err != nil {
		return err
	}
	c.Lock()
	defer c.Unlock()
	c.gauges[gaugeName] = gauge
	return nil
}

func (c *Int64GaugeHelper) Close(context.Context) {
	c.Lock()
	defer c.Unlock()
	clear(c.gauges)
}

type Float64HistogramHelper struct {
	sync.RWMutex
	histograms map[string]Float64Histogram
}

func NewFloat64HistogramHelper() *Float64HistogramHelper {
	return &Float64HistogramHelper{
		histograms: make(map[string]Float64Histogram),
	}
}

func (c *Float64HistogramHelper) Record(ctx context.Context, histogramName string, value float64, options ...metric.RecordOption) {
	c.RLock()
	defer c.RUnlock()
	histogram, ok := c.histograms[histogramName]
	if !ok {
		return
	}
	histogram.Record(ctx, value, options...)
}

func (c *Float64HistogramHelper) CreateHistogram(meter Meter, histogramName string, descriptions ...string) error {
	options := make([]metric.Float64HistogramOption, 0)
	if len(descriptions) > 0 && descriptions[0] != "" {
		options = append(options, metric.WithDescription(descriptions[0]))
	}
	gauge, err := meter.Float64Histogram(histogramName, options...)
	if err != nil {
		return err
	}
	c.Lock()
	defer c.Unlock()
	c.histograms[histogramName] = gauge
	return nil
}

func (c *Float64HistogramHelper) Close(context.Context) {
	c.Lock()
	defer c.Unlock()
	clear(c.histograms)
}

type Int64HistogramHelper struct {
	sync.RWMutex
	histograms map[string]Int64Histogram
}

func NewInt64HistogramHelper() *Int64HistogramHelper {
	return &Int64HistogramHelper{
		histograms: make(map[string]Int64Histogram),
	}
}

func (c *Int64HistogramHelper) Record(ctx context.Context, histogramName string, value int64, options ...metric.RecordOption) {
	c.RLock()
	defer c.RUnlock()
	histogram, ok := c.histograms[histogramName]
	if !ok {
		return
	}
	histogram.Record(ctx, value, options...)
}

func (c *Int64HistogramHelper) CreateHistogram(meter Meter, histogramName string, descriptions ...string) error {
	options := make([]metric.Int64HistogramOption, 0)
	if len(descriptions) > 0 && descriptions[0] != "" {
		options = append(options, metric.WithDescription(descriptions[0]))
	}
	gauge, err := meter.Int64Histogram(histogramName, options...)
	if err != nil {
		return err
	}
	c.Lock()
	defer c.Unlock()
	c.histograms[histogramName] = gauge
	return nil
}

func (c *Int64HistogramHelper) Close(context.Context) {
	c.Lock()
	defer c.Unlock()
	clear(c.histograms)
}

type Int64UpDownCounterHelper struct {
	sync.RWMutex
	counters map[string]Int64UpDownCounter
}

func NewInt64UpDownCounterHelper() *Int64UpDownCounterHelper {
	return &Int64UpDownCounterHelper{
		counters: make(map[string]Int64UpDownCounter),
	}
}

func (c *Int64UpDownCounterHelper) Add(ctx context.Context, counterName string, incr int64, options ...metric.AddOption) {
	c.RLock()
	defer c.RUnlock()
	counter, ok := c.counters[counterName]
	if !ok {
		return
	}
	counter.Add(ctx, incr, options...)
}

func (c *Int64UpDownCounterHelper) CreateCounter(meter Meter, counterName string, descriptions ...string) error {
	options := make([]metric.Int64UpDownCounterOption, 0)
	if len(descriptions) > 0 && descriptions[0] != "" {
		options = append(options, metric.WithDescription(descriptions[0]))
	}
	counter, err := meter.Int64UpDownCounter(counterName, options...)
	if err != nil {
		return err
	}
	c.Lock()
	defer c.Unlock()
	c.counters[counterName] = counter
	return nil
}

func (c *Int64UpDownCounterHelper) Close(context.Context) {
	c.Lock()
	defer c.Unlock()
	clear(c.counters)
}
