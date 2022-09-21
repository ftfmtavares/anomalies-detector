package collector

import (
	"reflect"
	"testing"
	"time"

	"github.com/ftfmtavares/anomalies-detector/config"
)

func Test_generateData(t *testing.T) {
	type args struct {
		metric    string
		dateStart time.Time
		dateEnd   time.Time
		timeStep  time.Duration
	}
	type wantArgs struct {
		length    int
		dateStart time.Time
	}

	timeRef := time.Now()

	tests := []struct {
		name string
		args args
		want wantArgs
	}{
		{
			name: "Data Generation with time steps of 1 day for a total of 5 days",
			args: args{
				metric:    "Revenue",
				dateStart: timeRef.AddDate(0, 0, -5),
				dateEnd:   timeRef,
				timeStep:  time.Duration(int64(time.Hour) * 24),
			},
			want: wantArgs{
				length:    5,
				dateStart: timeRef.AddDate(0, 0, -5),
			},
		},
		{
			name: "Data Generation with time steps of 1 hour for a total of 30 days",
			args: args{
				metric:    "Basket",
				dateStart: timeRef.AddDate(0, 0, -30),
				dateEnd:   timeRef,
				timeStep:  time.Duration(int64(time.Hour)),
			},
			want: wantArgs{
				length:    30 * 24,
				dateStart: timeRef.AddDate(0, 0, -30),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generateData(tt.args.metric, tt.args.dateStart, tt.args.dateEnd, tt.args.timeStep)

			//generateData returns random numbers which makes it impossible to define an expected exact result, so only the dataset length and time distribution are tested
			if len(got.AttributeData["Total"]) != tt.want.length {
				t.Errorf("len(generateData().AttributeData[\"Total\"] = %d, want %d", len(got.AttributeData["Total"]), tt.want.length)
			}
			for i, step := range got.AttributeData["Total"] {
				if step.Samples == 0 {
					t.Errorf("generateData().AttributeData[\"Total\"][%d].Samples = %d, want >0", i, step.Samples)
				}
				if step.Value == 0 {
					t.Errorf("generateData().AttributeData[\"Total\"][%d].Value = %f, want >0", i, step.Value)
				}
				if !step.DateStart.Equal(tt.want.dateStart.Add(tt.args.timeStep * time.Duration(i))) {
					t.Errorf("generateData().AttributeData[\"Total\"][%d].DateStart = %v, want %v", i, step.DateStart, tt.want.dateStart.Add(tt.args.timeStep*time.Duration(i)))
				}
			}
		})
	}
}

func Test_filterData(t *testing.T) {
	type args struct {
		metricData     MetricData
		collectFilters config.CollectFilters
	}

	timeRef := time.Now()

	tests := []struct {
		name string
		args args
		want MetricData
	}{
		{
			name: "Filter by minimum samples",
			args: args{
				metricData: MetricData{
					Metric:     "metric",
					Unit:       "unit",
					Attributes: []string{"Total", "Attribute1>Sub1", "Attribute1>Sub1>Sub1", "Attribute1>Sub1>Sub2", "Attribute1>Sub2", "Attribute2>Sub1", "Attribute2>Sub2"},
					AttributeData: map[string][]TimeStepData{
						"Total":                {{DateStart: timeRef, Value: 10, Samples: 100}},
						"Attribute1>Sub1":      {{DateStart: timeRef, Value: 10, Samples: 80}},
						"Attribute1>Sub1>Sub1": {{DateStart: timeRef, Value: 10, Samples: 50}},
						"Attribute1>Sub1>Sub2": {{DateStart: timeRef, Value: 10, Samples: 30}},
						"Attribute1>Sub2":      {{DateStart: timeRef, Value: 10, Samples: 20}},
						"Attribute2>Sub1":      {{DateStart: timeRef, Value: 10, Samples: 60}},
						"Attribute2>Sub2":      {{DateStart: timeRef, Value: 10, Samples: 40}},
					},
				},
				collectFilters: config.CollectFilters{
					MinVisitorsPerTimeStep: 90,
					AttributesFilterParams: map[string]config.FilterParams{},
				},
			},
			want: MetricData{
				Metric:     "metric",
				Unit:       "unit",
				Attributes: []string{"Total"},
				AttributeData: map[string][]TimeStepData{
					"Total": {{DateStart: timeRef, Value: 10, Samples: 100}},
				},
			},
		},
		{
			name: "Filter by limiting level of a given attribute",
			args: args{
				metricData: MetricData{
					Metric:     "metric",
					Unit:       "unit",
					Attributes: []string{"Total", "Attribute1>Sub1", "Attribute1>Sub1>Sub1", "Attribute1>Sub1>Sub2", "Attribute1>Sub2", "Attribute2>Sub1", "Attribute2>Sub2"},
					AttributeData: map[string][]TimeStepData{
						"Total":                {{DateStart: timeRef, Value: 10, Samples: 100}},
						"Attribute1>Sub1":      {{DateStart: timeRef, Value: 10, Samples: 80}},
						"Attribute1>Sub1>Sub1": {{DateStart: timeRef, Value: 10, Samples: 50}},
						"Attribute1>Sub1>Sub2": {{DateStart: timeRef, Value: 10, Samples: 30}},
						"Attribute1>Sub2":      {{DateStart: timeRef, Value: 10, Samples: 20}},
						"Attribute2>Sub1":      {{DateStart: timeRef, Value: 10, Samples: 60}},
						"Attribute2>Sub2":      {{DateStart: timeRef, Value: 10, Samples: 40}},
					},
				},
				collectFilters: config.CollectFilters{
					MinVisitorsPerTimeStep: 10,
					AttributesFilterParams: map[string]config.FilterParams{
						"Attribute1": {Level: 1, Top: 0},
					},
				},
			},
			want: MetricData{
				Metric:     "metric",
				Unit:       "unit",
				Attributes: []string{"Total", "Attribute1>Sub1", "Attribute1>Sub2", "Attribute2>Sub1", "Attribute2>Sub2"},
				AttributeData: map[string][]TimeStepData{
					"Total":           {{DateStart: timeRef, Value: 10, Samples: 100}},
					"Attribute1>Sub1": {{DateStart: timeRef, Value: 10, Samples: 80}},
					"Attribute1>Sub2": {{DateStart: timeRef, Value: 10, Samples: 20}},
					"Attribute2>Sub1": {{DateStart: timeRef, Value: 10, Samples: 60}},
					"Attribute2>Sub2": {{DateStart: timeRef, Value: 10, Samples: 40}},
				},
			},
		},
		{
			name: "Filter by filtering top values of given attributes",
			args: args{
				metricData: MetricData{
					Metric:     "metric",
					Unit:       "unit",
					Attributes: []string{"Total", "Attribute1>Sub1", "Attribute1>Sub1>Sub1", "Attribute1>Sub1>Sub2", "Attribute1>Sub2", "Attribute2>Sub1", "Attribute2>Sub2"},
					AttributeData: map[string][]TimeStepData{
						"Total":                {{DateStart: timeRef, Value: 10, Samples: 100}},
						"Attribute1>Sub1":      {{DateStart: timeRef, Value: 10, Samples: 80}},
						"Attribute1>Sub1>Sub1": {{DateStart: timeRef, Value: 10, Samples: 50}},
						"Attribute1>Sub1>Sub2": {{DateStart: timeRef, Value: 10, Samples: 30}},
						"Attribute1>Sub2":      {{DateStart: timeRef, Value: 10, Samples: 20}},
						"Attribute2>Sub1":      {{DateStart: timeRef, Value: 10, Samples: 60}},
						"Attribute2>Sub2":      {{DateStart: timeRef, Value: 10, Samples: 40}},
					},
				},
				collectFilters: config.CollectFilters{
					MinVisitorsPerTimeStep: 10,
					AttributesFilterParams: map[string]config.FilterParams{
						"Attribute1": {Level: 2, Top: 1},
						"Attribute2": {Level: 1, Top: 1},
					},
				},
			},
			want: MetricData{
				Metric:     "metric",
				Unit:       "unit",
				Attributes: []string{"Total", "Attribute1>Sub1", "Attribute1>Sub1>Sub1", "Attribute1>Sub2", "Attribute2>Sub1"},
				AttributeData: map[string][]TimeStepData{
					"Total":                {{DateStart: timeRef, Value: 10, Samples: 100}},
					"Attribute1>Sub1":      {{DateStart: timeRef, Value: 10, Samples: 80}},
					"Attribute1>Sub1>Sub1": {{DateStart: timeRef, Value: 10, Samples: 50}},
					"Attribute1>Sub2":      {{DateStart: timeRef, Value: 10, Samples: 20}},
					"Attribute2>Sub1":      {{DateStart: timeRef, Value: 10, Samples: 60}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterData(tt.args.metricData, tt.args.collectFilters); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("filterData() = %v, want %v", got, tt.want)
			}
		})
	}
}
