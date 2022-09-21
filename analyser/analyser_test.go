package analyser

import (
	"reflect"
	"testing"
	"time"

	"github.com/ftfmtavares/anomalies-detector/collector"
)

func TestDetectOutliers3Sigmas(t *testing.T) {
	type args struct {
		data                     []collector.TimeStepData
		PeriodEnd                time.Time
		outliersMultiplier       float64
		strongOutliersMultiplier float64
	}

	timeRef := time.Now()

	tests := []struct {
		name           string
		args           args
		wantedWarnings []eventPeriod
		wantedAlarms   []eventPeriod
		values         []float64
	}{
		{
			name:           "Samples with Z-Score >3 at samples #28-#29 and Z-score >2 at sample #30",
			args:           args{outliersMultiplier: 2, strongOutliersMultiplier: 3, PeriodEnd: timeRef},
			wantedWarnings: []eventPeriod{{outlierPeriodStart: timeRef.AddDate(0, 0, -1), outlierPeriodEnd: timeRef}},
			wantedAlarms:   []eventPeriod{{outlierPeriodStart: timeRef.AddDate(0, 0, -3), outlierPeriodEnd: timeRef.AddDate(0, 0, -1)}},
			values:         []float64{221, 254, 270, 264, 244, 241, 238, 243, 277, 237, 254, 289, 278, 264, 265, 243, 284, 244, 212, 242, 271, 243, 252, 230, 238, 214, 234, 1027, 1057, 911},
		},
		{
			name:           "Samples with Z-Score >3 at samples #28-#29 and Z-score >2 at sample #30",
			args:           args{outliersMultiplier: 3, strongOutliersMultiplier: 4, PeriodEnd: timeRef},
			wantedWarnings: []eventPeriod{{outlierPeriodStart: timeRef.AddDate(0, 0, -3), outlierPeriodEnd: timeRef.AddDate(0, 0, -1)}},
			wantedAlarms:   []eventPeriod{},
			values:         []float64{221, 254, 270, 264, 244, 241, 238, 243, 277, 237, 254, 289, 278, 264, 265, 243, 284, 244, 212, 242, 271, 243, 252, 230, 238, 214, 234, 1027, 1057, 911},
		},
	}

	for _, tt := range tests {
		tt.args.data = make([]collector.TimeStepData, len(tt.values))
		for i, val := range tt.values {
			tt.args.data[i].Samples = 100
			tt.args.data[i].DateStart = timeRef.AddDate(0, 0, -len(tt.values)+i)
			tt.args.data[i].Value = val
		}

		t.Run(tt.name, func(t *testing.T) {
			warnings, alarms := detectOutliers3Sigmas(tt.args.data, tt.args.PeriodEnd, tt.args.outliersMultiplier, tt.args.strongOutliersMultiplier)
			if !reflect.DeepEqual(warnings, tt.wantedWarnings) {
				t.Errorf("DetectOutliers3Sigmas() got = %v, want %v", warnings, tt.wantedWarnings)
			}
			if !reflect.DeepEqual(alarms, tt.wantedAlarms) {
				t.Errorf("DetectOutliers3Sigmas() got1 = %v, want %v", alarms, tt.wantedAlarms)
			}
		})
	}
}
