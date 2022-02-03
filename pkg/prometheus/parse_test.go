package prometheus

import (
	"fmt"
	"math"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

var units = []struct {
	in             *dto.MetricFamily
	extraLabels    map[string]string
	expectedMetric prometheus.Metric
	expectedError  error
}{
	{
		in: &dto.MetricFamily{
			Name: proto.String("name"),
			Help: proto.String("two-line\n doc  str\\ing"),
			Type: dto.MetricType_COUNTER.Enum(),
			Metric: []*dto.Metric{
				{
					Label: []*dto.LabelPair{
						{
							Name:  proto.String("labelname"),
							Value: proto.String("val1"),
						},
					},
					Counter: &dto.Counter{
						Value: proto.Float64(math.NaN()),
					},
				},
			},
		},
		extraLabels: map[string]string{
			"plugin": "test",
		},
		expectedMetric: prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				"name",
				"two-line\n doc  str\\ing",
				[]string{"labelname", "plugin"},
				nil,
			),
			prometheus.CounterValue,
			math.NaN(),
			"val1", "test",
		),
		expectedError: nil,
	},
	{
		in: &dto.MetricFamily{
			Name: proto.String("name2"),
			Help: proto.String("doc str\"ing 2"),
			Type: dto.MetricType_GAUGE.Enum(),
			Metric: []*dto.Metric{
				{
					Label: []*dto.LabelPair{
						{
							Name:  proto.String("labelname"),
							Value: proto.String("val2"),
						},
						{
							Name:  proto.String("basename"),
							Value: proto.String("basevalue2"),
						},
					},
					Gauge: &dto.Gauge{
						Value: proto.Float64(math.Inf(+1)),
					},
					TimestampMs: proto.Int64(54321),
				},
			},
		},
		extraLabels: map[string]string{},
		expectedMetric: prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				"name2",
				"doc str\"ing 2",
				[]string{"labelname", "basename"},
				nil,
			),
			prometheus.GaugeValue,
			math.Inf(+1),
			"val2", "basevalue2",
		),
		expectedError: nil,
	},
	{
		in: &dto.MetricFamily{
			Name: proto.String("my_summary"),
			Type: dto.MetricType_SUMMARY.Enum(),
			Metric: []*dto.Metric{
				{
					Label: []*dto.LabelPair{
						{
							Name:  proto.String("n1"),
							Value: proto.String("val1"),
						},
					},
					Summary: &dto.Summary{
						SampleCount: proto.Uint64(42),
						SampleSum:   proto.Float64(4711),
						Quantile: []*dto.Quantile{
							{
								Quantile: proto.Float64(0.5),
								Value:    proto.Float64(110),
							},
							{
								Quantile: proto.Float64(0.9),
								Value:    proto.Float64(140),
							},
						},
					},
					TimestampMs: proto.Int64(2),
				},
			},
		},
		extraLabels: map[string]string{},
		expectedMetric: prometheus.MustNewConstSummary(
			prometheus.NewDesc(
				"my_summary",
				"",
				[]string{"n1"},
				nil,
			),
			42,
			4711,
			map[float64]float64{
				0.5: 110,
				0.9: 140,
			},
			"val1",
		),
		expectedError: nil,
	},
	{
		in: &dto.MetricFamily{
			Name: proto.String("minimal_metric"),
			Type: dto.MetricType_UNTYPED.Enum(),
			Metric: []*dto.Metric{
				{
					Untyped: &dto.Untyped{
						Value: proto.Float64(1.234),
					},
				},
			},
		},
		extraLabels: map[string]string{},
		expectedMetric: prometheus.MustNewConstMetric(
			prometheus.NewDesc(
				"minimal_metric",
				"",
				[]string{},
				nil,
			),
			prometheus.UntypedValue,
			1.234,
		),
		expectedError: nil,
	},
	{
		in: &dto.MetricFamily{
			Name: proto.String("request_duration_microseconds"),
			Help: proto.String("The response latency."),
			Type: dto.MetricType_HISTOGRAM.Enum(),
			Metric: []*dto.Metric{
				{
					Histogram: &dto.Histogram{
						SampleCount: proto.Uint64(2693),
						SampleSum:   proto.Float64(1756047.3),
						Bucket: []*dto.Bucket{
							{
								UpperBound:      proto.Float64(100),
								CumulativeCount: proto.Uint64(123),
							},
							{
								UpperBound:      proto.Float64(120),
								CumulativeCount: proto.Uint64(412),
							},
							{
								UpperBound:      proto.Float64(144),
								CumulativeCount: proto.Uint64(592),
							},
							{
								UpperBound:      proto.Float64(172.8),
								CumulativeCount: proto.Uint64(1524),
							},
							{
								UpperBound:      proto.Float64(math.Inf(+1)),
								CumulativeCount: proto.Uint64(2693),
							},
						},
					},
				},
			},
		},
		extraLabels: map[string]string{},
		expectedMetric: prometheus.MustNewConstHistogram(
			prometheus.NewDesc(
				"request_duration_microseconds",
				"The response latency.",
				[]string{},
				nil,
			),
			2693,
			1756047.3,
			map[float64]uint64{
				100:          123,
				120:          412,
				144:          592,
				172.8:        1524,
				math.Inf(+1): 2693,
			},
		),
		expectedError: nil,
	},
}

func TestParseToMetric(t *testing.T) {
	for count, unit := range units {
		t.Run(fmt.Sprintf("test %d", count), func(t *testing.T) {
			out, err := ParsedToMetric(unit.extraLabels, unit.in)
			assert.Equal(t, unit.expectedError, err)
			if err != nil {
				assert.Equal(t, unit.expectedMetric.Desc(), out.Desc())

				// ideally we would deep inspect the 2 structures, for now comparing the textual represtation of the protobuf message is enough
				var m1, m2 dto.Metric
				assert.NoError(t, unit.expectedMetric.Write(&m1))
				assert.NoError(t, out.Write(&m2))

				assert.Equal(t, m1.String(), m2.String())
			}
		})
	}
}

func BenchmarkParseToMetric(b *testing.B) {
	for count, unit := range units {
		b.Run(fmt.Sprintf("benchmark %d", count), func(b *testing.B) {
			_, _ = ParsedToMetric(unit.extraLabels, unit.in)
		})
	}
}
