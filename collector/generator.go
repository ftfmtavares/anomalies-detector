package collector

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"
)

//Const block defines some mathematical parameters to be used on the data simulation
const (
	outlierDiffMultiplier            = 20.0
	attributeDivisionSampleDeviation = 0.2
	attributeDivisionValDeviation    = 0.4
	outlierProb                      = 0.001
	outlierMaxSize                   = 6
)

var (
	//List containing all supported metrics
	allMetrices = []string{
		"Revenue",
		"Basket",
		"Visits",
	}

	//Map that points to the respective units of supported metrics
	metricesUnits = map[string]string{
		"Revenue": "Total Orders (EUR)",
		"Basket":  "Average Basket Value (EUR)",
		"Visits":  "Number of Sessions",
	}

	//Metrics mathematical parameters to be used on the data simulation
	sampleCreationMetricsMap = map[string]sampleCreationMetricParams{
		"Revenue": {
			metricType:   "Sum",
			valStdDev:    20000,
			valMean:      100000,
			sampleStdDev: 300,
			sampleMean:   1500,
		},
		"Basket": {
			metricType:   "Average",
			valStdDev:    80,
			valMean:      400,
			sampleStdDev: 300,
			sampleMean:   1500,
		},
		"Visits": {
			metricType:   "Count",
			valStdDev:    4000,
			valMean:      20000,
			sampleStdDev: 4000,
			sampleMean:   20000,
		},
	}

	//Tree structure containing the attributes used on data simulation
	sampleCreationAttributesTree = []sampleCreationAttributeNode{
		{
			name: "DeviceType",
			subAttributes: []sampleCreationAttributeNode{
				{name: "Desktop", weight: 50},
				{name: "Tablet", weight: 10},
				{name: "Mobile", weight: 40},
			},
		},
		{
			name: "Browser",
			subAttributes: []sampleCreationAttributeNode{
				{name: "Chrome", weight: 50, subAttributes: []sampleCreationAttributeNode{
					{name: "v1", weight: 5},
					{name: "v2", weight: 15},
					{name: "v3", weight: 80}}},
				{name: "Edge", weight: 20},
				{name: "Firefox", weight: 10},
				{name: "Safari", weight: 20},
			},
		},
	}
)

//sampleCreationMetricParams is the structure that holds the metric mathematical parameters
type sampleCreationMetricParams struct {
	metricType   string
	valStdDev    float64
	valMean      float64
	sampleStdDev float64
	sampleMean   float64
}

//sampleCreationAttributeNodeis the node structure that holds the attributes parameteres
type sampleCreationAttributeNode struct {
	name          string
	weight        float64
	subAttributes []sampleCreationAttributeNode
}

//generateData simulates metrics data from e-commerce sites and returns it
//Input arguments define the metric and the data period while internal const and vars provide existing attributes and mathematical parameteres
//The simulation tries to create data as most realistic as possible following standard distributions and ocasional deviations in order to test the detection methods
func generateData(metric string, dateStart, dateEnd time.Time, timeStep time.Duration) MetricData {

	//Initializing the MetricData object to be returned
	metricData := MetricData{Metric: metric, Unit: metricesUnits[metric], Attributes: []string{}, AttributeData: map[string][]TimeStepData{}}

	//Calculating and allocating the time steps for the main total data (no attribute)
	metricData = allocMasterData(metricData, "Total", dateStart, dateEnd, timeStep)

	//Randomly generating standard distribution number of samples for the main total data (no attribute)
	fillMasterSamples(metricData.AttributeData["Total"], sampleCreationMetricsMap[metric])

	//Randomly adding deviations on the metric values for the main total data (no attribute)
	addMasterOutliers(metricData.AttributeData["Total"], sampleCreationMetricsMap[metric], outlierProb, outlierMaxSize, outlierDiffMultiplier)

	//Looping each main attribute
	for _, attributeNode := range sampleCreationAttributesTree {

		//Allocating and adding the time steps for all main attribute/sub-values combinations following the attributes tree recursively
		metricData = allocAttributesData(metricData, attributeNode, attributeNode.name, dateStart, dateEnd, timeStep)

		//Distributing main total number of samples through the several attribute/sub-values combinations following the attributes tree recursively
		metricData = splitSamples(metricData, attributeNode, metricData.AttributeData["Total"], attributeNode.name)

		//Randomly adding deviations on the metric values for all main attribute/sub-values combinations following the attributes tree recursively
		//Added deviations are then returned and added to the top layer attribute/sub-values combinations, including the main total
		if len(attributeNode.subAttributes) > 0 {
			var subOutliersInc []float64
			metricData, subOutliersInc = addAttributesOutliers(metricData, attributeNode, sampleCreationMetricsMap[metric], attributeNode.name, outlierProb/float64(len(attributeNode.subAttributes)), outlierMaxSize, outlierDiffMultiplier/2)
			for i := range metricData.AttributeData["Total"] {
				metricData.AttributeData["Total"][i].Value += subOutliersInc[i]
			}
		}
	}

	//Randomly generating standard distribution metric values for the main total data (no attribute)
	//The random standard distribution values are added to the existing deviations already generated
	fillMasterValues(metricData.AttributeData["Total"], sampleCreationMetricsMap[metric])

	//Looping each main attribute
	for _, attributeNode := range sampleCreationAttributesTree {

		//Distributing main total metric values through the several attribute/sub-values combinations following the attributes tree recursively
		//The random standard distribution values are added to the existing deviations already generated
		metricData = splitValues(metricData, attributeNode, metricData.AttributeData["Total"], sampleCreationMetricsMap[metric], attributeNode.name)
	}

	return metricData
}

//allocMasterData calculates and allocates the time steps for an isolated attribute
//Used for the main total data
func allocMasterData(metricData MetricData, path string, dateStart, dateEnd time.Time, stepDuration time.Duration) MetricData {
	newData := []TimeStepData{}
	dateStep := dateStart
	for dateStep.Before(dateEnd) {
		newTimeStepData := TimeStepData{DateStart: dateStep}
		newData = append(newData, newTimeStepData)
		dateStep = dateStep.Add(stepDuration)
	}
	metricData.Attributes = append(metricData.Attributes, path)
	metricData.AttributeData[path] = newData

	return metricData
}

//allocAttributesData calculates and allocates the time steps for all attribute/sub-values combinations following the given sampleCreationAttributeNode tree recursively
func allocAttributesData(metricData MetricData, node sampleCreationAttributeNode, path string, dateStart, dateEnd time.Time, stepDuration time.Duration) MetricData {
	for _, attribute := range node.subAttributes {
		newData := []TimeStepData{}
		dateStep := dateStart
		for dateStep.Before(dateEnd) {
			newTimeStepData := TimeStepData{DateStart: dateStep}
			newData = append(newData, newTimeStepData)
			dateStep = dateStep.Add(stepDuration)
		}
		newPath := fmt.Sprintf("%s>%s", path, attribute.name)
		metricData.Attributes = append(metricData.Attributes, newPath)
		metricData.AttributeData[newPath] = newData
		metricData = allocAttributesData(metricData, attribute, newPath, dateStart, dateEnd, stepDuration)
	}

	return metricData
}

//fillMasterSamples generates standard distribution number of samples for a given Time Step slice
//Used for the main total data
func fillMasterSamples(data []TimeStepData, metric sampleCreationMetricParams) {
	randSource := rand.NewSource(time.Now().UnixNano())
	randGen := rand.New(randSource)
	for i := range data {
		data[i].Samples = int(math.Round(randGen.NormFloat64()*metric.sampleStdDev + metric.sampleMean))
		if data[i].Samples < 0 {
			data[i].Samples = 0
		}
	}
}

//splitSamples distributes main total number of samples through all attribute/sub-values combinations following the given sampleCreationAttributeNode tree recursively
func splitSamples(metricData MetricData, node sampleCreationAttributeNode, masterData []TimeStepData, path string) MetricData {
	randSource := rand.NewSource(time.Now().UnixNano())
	randGen := rand.New(randSource)
	totalWeight := 0.0
	for _, subAttribute := range node.subAttributes {
		totalWeight += subAttribute.weight
	}
	for step := range masterData {
		remain := masterData[step].Samples
		for i := 0; i < len(node.subAttributes)-1; i++ {
			data := metricData.AttributeData[fmt.Sprintf("%s>%s", path, node.subAttributes[i].name)]
			weight := node.subAttributes[i].weight / totalWeight * (1 + randGen.Float64()*attributeDivisionSampleDeviation - attributeDivisionSampleDeviation/2)
			data[step].Samples = int(math.Round(weight * float64(masterData[step].Samples)))
			remain -= data[step].Samples
		}
		data := metricData.AttributeData[fmt.Sprintf("%s>%s", path, node.subAttributes[len(node.subAttributes)-1].name)]
		data[step].Samples = remain
	}
	for _, subAttribute := range node.subAttributes {
		if len(subAttribute.subAttributes) > 0 {
			metricData = splitSamples(metricData, subAttribute, metricData.AttributeData[fmt.Sprintf("%s>%s", path, subAttribute.name)], fmt.Sprintf("%s>%s", path, subAttribute.name))
		}
	}

	return metricData
}

//addMasterOutliers adds random deviations on the metric values for a given Time Step slice
//Used for the main total data
func addMasterOutliers(data []TimeStepData, metric sampleCreationMetricParams, outlierProb float64, outlierMaxSize int, outlierDiffMultiplier float64) {
	randSource := rand.NewSource(time.Now().UnixNano())
	randGen := rand.New(randSource)
	for step := 0; step < len(data); step++ {
		if randGen.Float64() < outlierProb {
			outlierDiff := outlierDiffMultiplier * metric.valStdDev
			if randGen.Float64() < 0.5 {
				outlierDiff *= -1
			}
			if metric.metricType == "Count" {
				outlierDiff = math.Round(outlierDiff)
			}
			outlierSize := randGen.Intn(outlierMaxSize) + 1
			if step+outlierSize > len(data)-1 {
				outlierSize = len(data) - step
			}

			log.Printf("Added Outlier - Total - %s <-> %s\n", data[step].DateStart.Format("2006-01-02 15:04"), data[step+outlierSize-1].DateStart.Format("2006-01-02 15:04"))

			for i := step; i < step+outlierSize; i++ {
				data[i].Value += outlierDiff
			}
			step += outlierSize - 1
		}
	}
}

//addAttributesOutliers adds random deviations on the metric values for all attribute/sub-values combinations following given sampleCreationAttributeNode tree recursively
//Added deviations are returned and added to the parent attribute/sub-values node
func addAttributesOutliers(metricData MetricData, node sampleCreationAttributeNode, metric sampleCreationMetricParams, path string, outlierProb float64, outlierMaxSize int, outlierDiffMultiplier float64) (MetricData, []float64) {
	randSource := rand.NewSource(time.Now().UnixNano())
	randGen := rand.New(randSource)

	topInc := make([]float64, len(metricData.AttributeData["Total"]))

	for _, subAttribute := range node.subAttributes {
		data := metricData.AttributeData[fmt.Sprintf("%s>%s", path, subAttribute.name)]
		for step := 0; step < len(data); step++ {
			if randGen.Float64() < outlierProb {
				outlierDiff := outlierDiffMultiplier * metric.valStdDev
				if randGen.Float64() < 0.5 {
					outlierDiff *= -1
				}
				if metric.metricType == "Count" {
					outlierDiff = math.Round(outlierDiff)
				}
				outlierSize := randGen.Intn(outlierMaxSize) + 1
				if step+outlierSize > len(data)-1 {
					outlierSize = len(data) - step
				}

				log.Printf("Added Outlier - %s>%s - %s <-> %s\n", path, subAttribute.name, data[step].DateStart.Format("2006-01-02 15:04"), data[step+outlierSize-1].DateStart.Format("2006-01-02 15:04"))

				for i := step; i < step+outlierSize; i++ {
					data[i].Value += outlierDiff
				}
				step += outlierSize - 1
			}
		}

		if len(subAttribute.subAttributes) > 0 {
			var subOutliersInc []float64
			metricData, subOutliersInc = addAttributesOutliers(metricData, subAttribute, metric, fmt.Sprintf("%s>%s", path, subAttribute.name), outlierProb/float64(len(node.subAttributes)), outlierMaxSize, outlierDiffMultiplier/2)
			for step := 0; step < len(data); step++ {
				data[step].Value += subOutliersInc[step]
			}
		}

		for step := 0; step < len(data); step++ {
			if data[step].Value != 0 {
				switch metric.metricType {
				case "Sum", "Count":
					topInc[step] += data[step].Value
				case "Average":
					totalSamples := 0.0
					for _, subAttribute := range node.subAttributes {
						totalSamples += float64(metricData.AttributeData[fmt.Sprintf("%s>%s", path, subAttribute.name)][step].Samples)
					}
					topInc[step] += data[step].Value * float64(data[step].Samples) / totalSamples
				}
			}
		}
	}

	return metricData, topInc
}

//fillMasterValues generates random standard distribution metric values for a given Time Step slice
//The random standard distribution values are added, not replacing the existing values
//Used for the main total data
func fillMasterValues(data []TimeStepData, metric sampleCreationMetricParams) {
	randSource := rand.NewSource(time.Now().UnixNano())
	randGen := rand.New(randSource)
	for i := range data {
		switch metric.metricType {
		case "Sum", "Average":
			data[i].Value += randGen.NormFloat64()*metric.valStdDev + metric.valMean
			if data[i].Value < 0 {
				data[i].Value = 0
			}
		case "Count":
			data[i].Samples += int(data[i].Value)
			if data[i].Samples < 0 {
				data[i].Samples = 0
			}
			data[i].Value = float64(data[i].Samples)
		}
	}
}

//splitValues distributes main total metric values through the several attribute/sub-values combinations following given sampleCreationAttributeNode tree recursively
//The random standard distribution values are added, not replacing the existing values
func splitValues(metricData MetricData, node sampleCreationAttributeNode, masterData []TimeStepData, metric sampleCreationMetricParams, path string) MetricData {
	randSource := rand.NewSource(time.Now().UnixNano())
	randGen := rand.New(randSource)
	for step := range masterData {
		switch metric.metricType {
		case "Sum":
			splitValue := masterData[step].Value
			for _, subAttribute := range node.subAttributes {
				data := metricData.AttributeData[fmt.Sprintf("%s>%s", path, subAttribute.name)]
				splitValue -= data[step].Value
			}
			remain := splitValue
			for i := 0; i < len(node.subAttributes)-1; i++ {
				data := metricData.AttributeData[fmt.Sprintf("%s>%s", path, node.subAttributes[i].name)]
				ratio := float64(data[step].Samples) / float64(masterData[step].Samples) * (1 + randGen.Float64()*attributeDivisionValDeviation - attributeDivisionValDeviation/2)
				partValue := ratio * splitValue
				data[step].Value += partValue
				if data[step].Value < 0 {
					data[step].Value = 0
				}
				remain -= partValue
			}
			data := metricData.AttributeData[fmt.Sprintf("%s>%s", path, node.subAttributes[len(node.subAttributes)-1].name)]
			data[step].Value += remain
			if data[step].Value < 0 {
				data[step].Value = 0
			}
		case "Average":
			splitValue := masterData[step].Value
			for _, subAttribute := range node.subAttributes {
				data := metricData.AttributeData[fmt.Sprintf("%s>%s", path, subAttribute.name)]
				splitValue -= data[step].Value * float64(data[step].Samples) / float64(masterData[step].Samples)
			}
			remain := splitValue
			for i := 0; i < len(node.subAttributes)-1; i++ {
				data := metricData.AttributeData[fmt.Sprintf("%s>%s", path, node.subAttributes[i].name)]
				ratio := 1 + randGen.Float64()*attributeDivisionValDeviation - attributeDivisionValDeviation/2
				partValue := ratio * splitValue
				data[step].Value += partValue
				if data[step].Value < 0 {
					data[step].Value = 0
				}
				remain -= partValue * float64(data[step].Samples) / float64(masterData[step].Samples)
			}
			data := metricData.AttributeData[fmt.Sprintf("%s>%s", path, node.subAttributes[len(node.subAttributes)-1].name)]
			data[step].Value += remain * float64(masterData[step].Samples) / float64(data[step].Samples)
			if data[step].Value < 0 {
				data[step].Value = 0
			}
		case "Count":
			splitValue := masterData[step].Value
			originalSamples := 0
			for _, subAttribute := range node.subAttributes {
				data := metricData.AttributeData[fmt.Sprintf("%s>%s", path, subAttribute.name)]
				splitValue -= data[step].Value
				originalSamples += data[step].Samples
			}
			remain := splitValue
			for i := 0; i < len(node.subAttributes)-1; i++ {
				data := metricData.AttributeData[fmt.Sprintf("%s>%s", path, node.subAttributes[i].name)]
				ratio := float64(data[step].Samples) / float64(originalSamples)
				partValue := math.Round(ratio * splitValue)
				data[step].Value += partValue
				if data[step].Value < 0 {
					data[step].Value = 0
				}
				data[step].Samples = int(data[step].Value)
				remain -= partValue
			}
			data := metricData.AttributeData[fmt.Sprintf("%s>%s", path, node.subAttributes[len(node.subAttributes)-1].name)]
			data[step].Value += remain
			if data[step].Value < 0 {
				data[step].Value = 0
			}
			data[step].Samples = int(data[step].Value)
		}
	}
	for _, subAttribute := range node.subAttributes {
		if len(subAttribute.subAttributes) > 0 {
			metricData = splitValues(metricData, subAttribute, metricData.AttributeData[fmt.Sprintf("%s>%s", path, subAttribute.name)], metric, fmt.Sprintf("%s>%s", path, subAttribute.name))
		}
	}

	return metricData
}
