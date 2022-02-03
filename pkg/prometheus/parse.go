package prometheus

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
)

func ParsedToMetric(labels map[string]string, raw *io_prometheus_client.MetricFamily) (prometheus.Metric, error) {
	if raw.Type == nil {
		return nil, errors.New("Invalid metric type?")
	}

	switch *raw.Type {
	case io_prometheus_client.MetricType_COUNTER:
		return parseCounter(labels, raw)
	case io_prometheus_client.MetricType_GAUGE:
		return parseGauge(labels, raw)
	case io_prometheus_client.MetricType_SUMMARY:
		return parseSummary(labels, raw)
	case io_prometheus_client.MetricType_UNTYPED:
		return parseUntyped(labels, raw)
	case io_prometheus_client.MetricType_HISTOGRAM:
		return parseHistogram(labels, raw)
	default:
		logrus.Debugf("Unknown type: %+v", raw)
	}

	panic("This shouldn't be reachable..")
}

func parseLabelDefinitions(labels map[string]string, raw *io_prometheus_client.Metric) []string {
	out := make([]string, 0)

	for key := range labels {
		out = append(out, key)
	}

	for _, label := range raw.Label {
		out = append(out, label.GetName())
	}

	return out
}

func parseLabelValues(order []string, labels map[string]string, raw *io_prometheus_client.Metric) []string {
	// prometheus expects the values in the same order as keys
	// sadly iterating over a map in golang does NOT always mean
	// the results are returned in the same order, so instead we pass
	// both the extra label values (labels) and the order that we want everything in.

	out := make([]string, 0)

	for _, key := range order {
		value, ok := labels[key]
		if ok {
			out = append(out, value)
		} else {
			for _, label := range raw.Label {
				if label.GetName() == key {
					out = append(out, label.GetValue())
					break
				}
			}
		}
	}

	return out
}

func parseCounter(extraLabels map[string]string, raw *io_prometheus_client.MetricFamily) (prometheus.Metric, error) {
	labels := parseLabelDefinitions(extraLabels, raw.Metric[0])
	return prometheus.NewConstMetric(
		prometheus.NewDesc(
			raw.GetName(),
			raw.GetHelp(),
			labels,
			nil),
		prometheus.CounterValue,
		raw.Metric[0].Counter.GetValue(),
		parseLabelValues(labels, extraLabels, raw.Metric[0])...,
	)
}

func parseGauge(extraLabels map[string]string, raw *io_prometheus_client.MetricFamily) (prometheus.Metric, error) {
	labels := parseLabelDefinitions(extraLabels, raw.Metric[0])
	return prometheus.NewConstMetric(
		prometheus.NewDesc(
			raw.GetName(),
			raw.GetHelp(),
			labels,
			nil),
		prometheus.GaugeValue,
		raw.Metric[0].Gauge.GetValue(),
		parseLabelValues(labels, extraLabels, raw.Metric[0])...,
	)
}

func parseQuantiles(summary *io_prometheus_client.Summary) (map[float64]float64, error) {
	out := make(map[float64]float64)

	for _, value := range summary.Quantile {
		out[value.GetQuantile()] = value.GetValue()
	}

	return out, nil
}

func parseSummary(extraLabels map[string]string, raw *io_prometheus_client.MetricFamily) (prometheus.Metric, error) {
	labels := parseLabelDefinitions(extraLabels, raw.Metric[0])
	quantiles, err := parseQuantiles(raw.Metric[0].Summary)
	if err != nil {
		return nil, err
	}

	return prometheus.NewConstSummary(
		prometheus.NewDesc(
			raw.GetName(),
			raw.GetHelp(),
			labels,
			nil),
		raw.Metric[0].Summary.GetSampleCount(),
		raw.Metric[0].Summary.GetSampleSum(),
		quantiles,
		parseLabelValues(labels, extraLabels, raw.Metric[0])...,
	)
}

func parseUntyped(extraLabels map[string]string, raw *io_prometheus_client.MetricFamily) (prometheus.Metric, error) {
	labels := parseLabelDefinitions(extraLabels, raw.Metric[0])
	return prometheus.NewConstMetric(
		prometheus.NewDesc(
			raw.GetName(),
			raw.GetHelp(),
			labels,
			nil,
		),
		prometheus.UntypedValue,
		raw.Metric[0].Untyped.GetValue(),
		parseLabelValues(labels, extraLabels, raw.Metric[0])...,
	)
}

func parseBuckets(histogram *io_prometheus_client.Histogram) (map[float64]uint64, error) {
	out := make(map[float64]uint64)

	for _, value := range histogram.Bucket {
		out[value.GetUpperBound()] = value.GetCumulativeCount()
	}

	return out, nil
}

func parseHistogram(extraLabels map[string]string, raw *io_prometheus_client.MetricFamily) (prometheus.Metric, error) {
	labels := parseLabelDefinitions(extraLabels, raw.Metric[0])
	buckets, err := parseBuckets(raw.Metric[0].Histogram)
	if err != nil {
		return nil, err
	}

	return prometheus.NewConstHistogram(
		prometheus.NewDesc(
			raw.GetName(),
			raw.GetHelp(),
			labels,
			nil,
		),
		raw.Metric[0].Histogram.GetSampleCount(),
		raw.Metric[0].Histogram.GetSampleSum(),
		buckets,
		parseLabelValues(labels, extraLabels, raw.Metric[0])...,
	)
}
