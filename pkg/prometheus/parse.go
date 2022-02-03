package prometheus

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
)

func ParsedToMetric(labels map[string]string, raw *io_prometheus_client.MetricFamily) ([]prometheus.Metric, error) {
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

func parseCounter(extraLabels map[string]string, raw *io_prometheus_client.MetricFamily) ([]prometheus.Metric, error) {
	out := make([]prometheus.Metric, 0)

	for _, metric := range raw.Metric {
		labels := parseLabelDefinitions(extraLabels, metric)
		m, err := prometheus.NewConstMetric(
			prometheus.NewDesc(
				raw.GetName(),
				raw.GetHelp(),
				labels,
				nil),
			prometheus.CounterValue,
			metric.Counter.GetValue(),
			parseLabelValues(labels, extraLabels, metric)...,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}

	return out, nil
}

func parseGauge(extraLabels map[string]string, raw *io_prometheus_client.MetricFamily) ([]prometheus.Metric, error) {
	out := make([]prometheus.Metric, 0)

	for _, metric := range raw.Metric {
		labels := parseLabelDefinitions(extraLabels, metric)
		m, err := prometheus.NewConstMetric(
			prometheus.NewDesc(
				raw.GetName(),
				raw.GetHelp(),
				labels,
				nil),
			prometheus.GaugeValue,
			metric.Counter.GetValue(),
			parseLabelValues(labels, extraLabels, metric)...,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}

	return out, nil
}

func parseQuantiles(summary *io_prometheus_client.Summary) (map[float64]float64, error) {
	out := make(map[float64]float64)

	for _, value := range summary.Quantile {
		out[value.GetQuantile()] = value.GetValue()
	}

	return out, nil
}

func parseSummary(extraLabels map[string]string, raw *io_prometheus_client.MetricFamily) ([]prometheus.Metric, error) {
	out := make([]prometheus.Metric, 0)

	for _, metric := range raw.Metric {
		labels := parseLabelDefinitions(extraLabels, metric)
		quantiles, err := parseQuantiles(metric.Summary)
		if err != nil {
			return nil, err
		}

		m, err := prometheus.NewConstSummary(
			prometheus.NewDesc(
				raw.GetName(),
				raw.GetHelp(),
				labels,
				nil),
			metric.Summary.GetSampleCount(),
			metric.Summary.GetSampleSum(),
			quantiles,
			parseLabelValues(labels, extraLabels, metric)...,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}

	return out, nil
}

func parseUntyped(extraLabels map[string]string, raw *io_prometheus_client.MetricFamily) ([]prometheus.Metric, error) {
	out := make([]prometheus.Metric, 0)

	for _, metric := range raw.Metric {
		labels := parseLabelDefinitions(extraLabels, metric)
		m, err := prometheus.NewConstMetric(
			prometheus.NewDesc(
				raw.GetName(),
				raw.GetHelp(),
				labels,
				nil),
			prometheus.UntypedValue,
			metric.Counter.GetValue(),
			parseLabelValues(labels, extraLabels, metric)...,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}

	return out, nil
}

func parseBuckets(histogram *io_prometheus_client.Histogram) (map[float64]uint64, error) {
	out := make(map[float64]uint64)

	for _, value := range histogram.Bucket {
		out[value.GetUpperBound()] = value.GetCumulativeCount()
	}

	return out, nil
}

func parseHistogram(extraLabels map[string]string, raw *io_prometheus_client.MetricFamily) ([]prometheus.Metric, error) {
	out := make([]prometheus.Metric, 0)

	for _, metric := range raw.Metric {
		labels := parseLabelDefinitions(extraLabels, metric)
		buckets, err := parseBuckets(metric.Histogram)
		if err != nil {
			return nil, err
		}

		m, err := prometheus.NewConstHistogram(
			prometheus.NewDesc(
				raw.GetName(),
				raw.GetHelp(),
				labels,
				nil,
			),
			metric.Histogram.GetSampleCount(),
			metric.Histogram.GetSampleSum(),
			buckets,
			parseLabelValues(labels, extraLabels, metric)...,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}

	return out, nil
}
