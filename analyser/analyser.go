package analyser

import (
	"log"
	"math"
	"time"

	"github.com/ftfmtavares/anomalies-detector/collector"
	"github.com/ftfmtavares/anomalies-detector/config"
)

//OutlierReport provides the structure to store all detected outliers of a given site
type OutlierReport struct {
	SiteId                  string         `json:"siteId"`
	OutliersDetectionMethod string         `json:"outliersDetectionMethod"`
	CheckDateStart          time.Time      `json:"checkTimeStart"`
	CheckDateEnd            time.Time      `json:"checkTimeEnd"`
	TimeAgo                 string         `json:"timeAgo"`
	TimeStep                string         `json:"timeStep"`
	DateStart               time.Time      `json:"dateStart"`
	DateEnd                 time.Time      `json:"dateEnd"`
	Result                  OutlierResults `json:"result"`
}

//OutlierResults holds the list of detected warnings and alarms
type OutlierResults struct {
	Warnings []OutlierEvent `json:"warnings"`
	Alarms   []OutlierEvent `json:"alarms"`
}

//OutlierEvent provides the structure to store the warning or alarm details
type OutlierEvent struct {
	OutlierPeriodStart time.Time `json:"outlierPeriodStart"`
	OutlierPeriodEnd   time.Time `json:"outlierPeriodEnd"`
	Metric             string    `json:"metric"`
	Attribute          string    `json:"attribute"`
}

//eventPeriod provides the structure to store a period of time
type eventPeriod struct {
	outlierPeriodStart time.Time
	outlierPeriodEnd   time.Time
}

//GetResults takes the entire data from a site and the respective configurations in order to look for outliers
//An OutlierReport is generated and returned
func GetResults(siteData collector.SiteData, dataConf config.Dataset, methodParams config.DetectionMethodsParams) OutlierReport {

	//Initalizing the resulting OutlierReport logging the check date start at the same time
	res := OutlierReport{
		SiteId:                  siteData.SiteId,
		OutliersDetectionMethod: dataConf.OutliersDetectionMethod,
		CheckDateStart:          time.Now(),
		TimeAgo:                 dataConf.TimeAgo,
		TimeStep:                dataConf.TimeStep,
		DateStart:               siteData.DateStart,
		DateEnd:                 siteData.DateEnd,
		Result: OutlierResults{
			Warnings: []OutlierEvent{},
			Alarms:   []OutlierEvent{},
		},
	}

	//Looping all attribute/sub-values combinations of each metric
	for _, metricData := range siteData.Metrics {
		for _, attribute := range metricData.Attributes {
			var warnings []eventPeriod
			var alarms []eventPeriod

			//Checking which detection method should be used and call the respective function
			switch res.OutliersDetectionMethod {
			case "3-sigmas":
				warnings, alarms = detectOutliers3Sigmas(metricData.AttributeData[attribute], siteData.DateEnd, methodParams.ThreeSigmas.OutliersMultiplier, methodParams.ThreeSigmas.StrongOutliersMultiplier)
			default:
				log.Printf("Detection Method %s not implemented\n", res.OutliersDetectionMethod)
				warnings = []eventPeriod{}
				alarms = []eventPeriod{}
			}

			//Taking the returned event periods and creating the respective warnings and alarms on the report
			for _, warning := range warnings {
				newOutlierEvent := OutlierEvent{
					OutlierPeriodStart: warning.outlierPeriodStart,
					OutlierPeriodEnd:   warning.outlierPeriodEnd,
					Metric:             metricData.Metric,
					Attribute:          attribute,
				}
				res.Result.Warnings = append(res.Result.Warnings, newOutlierEvent)
			}
			for _, alarm := range alarms {
				newOutlierEvent := OutlierEvent{
					OutlierPeriodStart: alarm.outlierPeriodStart,
					OutlierPeriodEnd:   alarm.outlierPeriodEnd,
					Metric:             metricData.Metric,
					Attribute:          attribute,
				}
				res.Result.Alarms = append(res.Result.Alarms, newOutlierEvent)
			}
		}
	}

	//Closing the log time just before returning the report
	res.CheckDateEnd = time.Now()
	return res
}

//detectOutliers3Sigmas implements the 3-sigmas method
//It takes the time step data and the method parameters as inputs and returns 2 event periods list containg the detected warnings and alarms
func detectOutliers3Sigmas(data []collector.TimeStepData, PeriodEnd time.Time, outliersMultiplier, strongOutliersMultiplier float64) ([]eventPeriod, []eventPeriod) {
	count := len(data)
	sum := 0.0
	mean := 0.0
	sd := 0.0

	//1st loop to calculate Sum and Mean
	for _, stepData := range data {
		sum += stepData.Value
	}
	mean = sum / float64(count)

	//2nd loop to calculate Standard Deviation
	for _, stepData := range data {
		sd += math.Pow(stepData.Value-mean, 2)
	}
	sd = math.Sqrt(sd / float64(count))

	//Calculating the Z-Score limits for warnings and alarms
	strongLimit := strongOutliersMultiplier * sd
	weakLimit := outliersMultiplier * sd

	//Initializing the resulting event periods
	warnings := []eventPeriod{}
	alarms := []eventPeriod{}

	//3rd loop to identify metric values that fall above the warning or alarm Z-score limits
	//A state machine keeps track if the beginning of an event period has been detected already and if it's an alarm or warning
	beginStep := -1
	strongEvent := false
	for ind := 0; ind < len(data); ind++ {

		//Z-Score above alarm limit
		//If no event was previously detected, it registers the start of a new alarm period
		//If a warning start was previously detected, it closes the warning and registers the start of a new alarm period
		//If an alarm start was previously detected, it does nothing and proceeds within the loop
		if math.Abs(data[ind].Value-mean) > strongLimit {
			if beginStep == -1 {
				beginStep = ind
				strongEvent = true
			} else if !strongEvent {
				newEvent := eventPeriod{
					outlierPeriodStart: data[beginStep].DateStart,
					outlierPeriodEnd:   data[ind].DateStart,
				}
				warnings = append(warnings, newEvent)
				beginStep = ind
				strongEvent = true
			}

			//Z-Score above warning limit
			//If no event was previously detected, it registers the start of a new warning period
			//If a warning start was previously detected, it does nothing and proceeds within the loop
			//If an alarm start was previously detected, it closes the alarm and registers the start of a new warning period
		} else if math.Abs(data[ind].Value-mean) > weakLimit {
			if beginStep == -1 {
				beginStep = ind
				strongEvent = false
			} else if strongEvent {
				newEvent := eventPeriod{
					outlierPeriodStart: data[beginStep].DateStart,
					outlierPeriodEnd:   data[ind].DateStart,
				}
				alarms = append(alarms, newEvent)
				beginStep = ind
				strongEvent = false
			}

			//Z-Score normal
			//If no event was previously detected, it does nothing and proceeds within the loop
			//If a warning start was previously detected, it closes it
			//If an alarm start was previously detected, it closes it
		} else {
			if beginStep != -1 {
				newEvent := eventPeriod{
					outlierPeriodStart: data[beginStep].DateStart,
					outlierPeriodEnd:   data[ind].DateStart,
				}
				if strongEvent {
					alarms = append(alarms, newEvent)
				} else {
					warnings = append(warnings, newEvent)
				}
				beginStep = -1
			}
		}
	}

	//Closing any detected event still open in the end of the loop
	if beginStep != -1 {
		newEvent := eventPeriod{
			outlierPeriodStart: data[beginStep].DateStart,
			outlierPeriodEnd:   PeriodEnd,
		}
		if strongEvent {
			alarms = append(alarms, newEvent)
		} else {
			warnings = append(warnings, newEvent)
		}
	}

	return warnings, alarms
}
