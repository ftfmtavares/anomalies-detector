package collector

import (
	"log"
	"strings"
	"time"

	"github.com/ftfmtavares/anomalies-detector/config"
	"github.com/ftfmtavares/anomalies-detector/utils"
)

//SiteData provides the structure to store all the collected data of a given site
type SiteData struct {
	SiteId    string       `json:"siteId"`
	DateStart time.Time    `json:"dateStart"`
	DateEnd   time.Time    `json:"dateEnd"`
	Metrics   []MetricData `json:"metrics"`
}

//MetricData contains all collected data for each metric of a given site
//Attributes field contains an ordered list of all attributes and sub-values combinations
//AttributeData field is a map that points to a slice of TimeStepData of the respective attribute/sub-values combination
type MetricData struct {
	Metric        string                    `json:"metric"`
	Unit          string                    `json:"unit"`
	Attributes    []string                  `json:"attributes"`
	AttributeData map[string][]TimeStepData `json:"attributeData"`
}

//GetSamplesCount is a method of MetricData that returns the total samples count of a given attribute/sub-values combination
//For this exercise, the calculation is run for each request but additional implementations can be done to MetricData in order to protect and store this calculation
func (metricData MetricData) GetSamplesCount(attribute string) int {
	sum := 0
	for _, stepData := range metricData.AttributeData[attribute] {
		sum += stepData.Samples
	}
	return sum
}

//GetLevel is a method of MetricData that returns the depth of a given attribute/sub-values combination
//For this exercise, the calculation is run for each request but additional implementations can be done to MetricData in order to protect and store this calculation
func (metricData MetricData) GetLevel(attribute string) int {
	return strings.Count(attribute, ">")
}

//GetLevel is a method of MetricData that returns the rank of a given attribute/sub-values combination in comparison to its peers
//Rank is calculated by comparing the number of samples from higher to lower while in case of equal number, rank is defined by alphabetical order
//For this exercise, the calculation is run for each request but additional implementations can be done to MetricData in order to protect and store this calculation
func (metricData MetricData) GetRank(attribute string) int {
	prefix := ""
	pathParts := strings.Split(attribute, ">")
	if len(pathParts) > 0 {
		prefix = strings.Join(pathParts[:len(pathParts)-1], ">")
	}
	attributeSamples := metricData.GetSamplesCount(attribute)

	rank := 1
	for _, compareAttribute := range metricData.Attributes {
		compareAttributeSamples := metricData.GetSamplesCount(compareAttribute)
		if compareAttribute != attribute && compareAttribute != prefix && strings.HasPrefix(compareAttribute, prefix) && (compareAttributeSamples > attributeSamples || (compareAttributeSamples == attributeSamples && compareAttribute < attribute)) {
			rank++
		}
	}

	return rank
}

//TimeStepData represents the data of a single time step
type TimeStepData struct {
	DateStart time.Time `json:"dateStart"`
	Value     float64   `json:"value"`
	Samples   int       `json:"samples"`
}

//GetData takes a site configuration and returns the respective data
func GetData(dataSet config.Dataset) SiteData {

	//Converting time periods in string format to be used as time.Duration
	timeAgoDuration, err := utils.StrToDuration(dataSet.TimeAgo)
	if err != nil {
		log.Panic(err)
	}
	timeStepDuration, err := utils.StrToDuration(dataSet.TimeStep)
	if err != nil {
		log.Panic(err)
	}

	//Initializing the siteData object to be returned
	siteData := SiteData{SiteId: dataSet.SiteId}
	siteData.DateEnd = time.Now()
	siteData.DateStart = siteData.DateEnd.Add(-1 * timeAgoDuration)
	siteData.Metrics = []MetricData{}

	//If the configured metric is "all", a list with all supported metrics will be used instead
	var coveredMetrics []string
	if len(dataSet.MetricesList) > 0 && strings.ToLower(dataSet.MetricesList[0]) == "all" {
		coveredMetrics = allMetrices
	} else {
		coveredMetrics = dataSet.MetricesList
	}

	//Looping all selected metrics
	for _, metric := range coveredMetrics {
		log.Printf("Getting Data - %s - %s\n", dataSet.SiteId, metric)

		//Since there is no access to the repository at this stage, data generation methods are used instead
		//Attribute filters would be applied while accessing and reading the repository but for now, they are applied in a separate call
		metricData := generateData(metric, siteData.DateStart, siteData.DateEnd, timeStepDuration)
		metricData = filterData(metricData, *dataSet.SiteCollectFilters)

		//Adds the read metric data to the result
		siteData.Metrics = append(siteData.Metrics, metricData)
	}

	return siteData
}

//filterData checks data from all attribute/sub-values combinations and removes those that don't meet the configured filters
func filterData(metricData MetricData, collectFilters config.CollectFilters) MetricData {

	//Calculating total minimum samples for the given period
	minSamples := collectFilters.MinVisitorsPerTimeStep * len(metricData.AttributeData["Total"])

	//Initializing a slice to hold the removal indication of each data set
	toRemove := make([]bool, len(metricData.Attributes))

	//Looping all existing attribute/sub-values combinations
	for ind, attribute := range metricData.Attributes {

		//Calculating the number of samples, atribute depth and number of samples rank in comparison with its peers
		samples := metricData.GetSamplesCount(attribute)
		level := metricData.GetLevel(attribute)
		rank := metricData.GetRank(attribute)

		//Spliting the path in order to isolate the main attribute name
		pathParts := strings.Split(attribute, ">")

		//Comparing the dataset attribute depth and check with existing filter
		if collectFilters.AttributesFilterParams[pathParts[0]].Level != 0 && collectFilters.AttributesFilterParams[pathParts[0]].Level < level {
			log.Printf("Filtering %s - Level %d higher than limit %d\n", attribute, level, collectFilters.AttributesFilterParams[pathParts[0]].Level)
			toRemove[ind] = true
		}

		//Comparing the dataset rank and check with existing filter
		if collectFilters.AttributesFilterParams[pathParts[0]].Level != 0 && collectFilters.AttributesFilterParams[pathParts[0]].Level == level && collectFilters.AttributesFilterParams[pathParts[0]].Top != 0 && collectFilters.AttributesFilterParams[pathParts[0]].Top < rank {
			log.Printf("Filtering %s - Rank %d not in top %d\n", attribute, rank, collectFilters.AttributesFilterParams[pathParts[0]].Top)
			toRemove[ind] = true
		}

		//Comparing the number of samples with total minimum
		if samples < minSamples {
			log.Printf("Filtering %s - Samples %d less than min %d\n", attribute, samples, minSamples)
			toRemove[ind] = true
		}
	}

	//Removing all identified datasets from the list
	for ind := len(metricData.Attributes) - 1; ind >= 0; ind-- {
		if toRemove[ind] {
			delete(metricData.AttributeData, metricData.Attributes[ind])
			if ind == len(metricData.Attributes)-1 {
				metricData.Attributes = metricData.Attributes[:ind]
			} else {
				metricData.Attributes = append(metricData.Attributes[:ind], metricData.Attributes[ind+1:]...)
			}
		}
	}

	return metricData
}
