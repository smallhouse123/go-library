package metrics

import (
	"errors"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/fx"
)

var (
	Service = fx.Provide(New)
)

type Params struct {
	fx.In

	ServiceName string `name:"serviceName"`
}

type PromMetric struct {
	service            string
	histogramCollector sync.Map
	counterCollector   sync.Map
	mutex              sync.Mutex
}

func New(p Params) Metrics {
	return &PromMetric{
		service:            p.ServiceName,
		histogramCollector: sync.Map{},
		mutex:              sync.Mutex{},
	}
}

func (p *PromMetric) BumpTime(key string, tags ...string) (Endable, error) {
	if len(tags)%2 != 0 {
		return nil, errors.New("tags must be a multiplier of 2")
	}

	id := p.service + key

	// First check without a lock
	if collector, ok := p.histogramCollector.Load(id); ok {
		duration := collector.(*prometheus.HistogramVec)
		labels := tagsToLabels(tags)
		timer := prometheus.NewTimer(duration.With(labels))
		return &promTimer{
			timer: timer,
		}, nil
	}

	// Lock to handle concurrent registrations
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Double-check after acquiring the lock
	if collector, ok := p.histogramCollector.Load(id); ok {
		duration := collector.(*prometheus.HistogramVec)
		labels := tagsToLabels(tags)
		timer := prometheus.NewTimer(duration.With(labels))
		return &promTimer{
			timer: timer,
		}, nil
	}

	// Create and register the new metric
	promOpts := prometheus.HistogramOpts{
		Namespace: p.service,
		Name:      key,
	}

	keyArr, _ := tagsToKeyAndVals(tags)
	labels := tagsToLabels(tags)

	duration := prometheus.NewHistogramVec(promOpts, keyArr)
	if err := prometheus.Register(duration); err != nil {
		return nil, err
	}

	// Store the metric in the map
	p.histogramCollector.Store(id, duration)

	// Start the timer
	timer := prometheus.NewTimer(duration.With(labels))
	return &promTimer{
		timer: timer,
	}, nil
}

func tagsToKeyAndVals(tags []string) ([]string, []string) {
	keyArr := []string{}
	valArr := []string{}

	if tags == nil {
		return keyArr, valArr
	}

	if len(tags)%2 != 0 {
		return keyArr, valArr
	}

	for i := 0; i < len(tags); i += 2 {
		key := tags[i]
		val := tags[i+1]
		keyArr = append(keyArr, key)
		valArr = append(valArr, val)
	}

	return keyArr, valArr
}

func tagsToLabels(tags []string) prometheus.Labels {
	newLabels := prometheus.Labels{}

	for i := 0; i < len(tags); i += 2 {
		key := tags[i]
		val := tags[i+1]
		newLabels[key] = val
	}
	return newLabels
}

type promTimer struct {
	timer *prometheus.Timer
}

func (p *promTimer) End() {
	p.timer.ObserveDuration()
}

func (p *PromMetric) BumpCount(key string, val float64, tags ...string) error {
	if len(tags)%2 != 0 {
		return errors.New("tags must be a multiplier of 2")
	}

	id := p.service + key

	// First check without a lock
	if collector, ok := p.counterCollector.Load(id); ok {
		counter := collector.(*prometheus.CounterVec)
		labels := tagsToLabels(tags)
		counter.With(labels).Add(val)
		return nil
	}

	// Lock to handle concurrent registrations
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Double-check after acquiring the lock
	if collector, ok := p.counterCollector.Load(id); ok {
		counter := collector.(*prometheus.CounterVec)
		labels := tagsToLabels(tags)
		counter.With(labels).Add(val)
		return nil
	}

	// Create and register the new metric
	promOpts := prometheus.CounterOpts{
		Namespace: p.service,
		Name:      key,
	}

	keyArr, _ := tagsToKeyAndVals(tags)

	counter := prometheus.NewCounterVec(promOpts, keyArr)
	if err := prometheus.Register(counter); err != nil {
		return err
	}

	// Store the metric in the map
	p.counterCollector.Store(id, counter)

	// Increment the counter with the given value
	labels := tagsToLabels(tags)
	counter.With(labels).Add(val)

	return nil
}
