package config

import (
	"encoding/json"
	"log"
	"os"
)

//ApplicationConfig provides the structure for the entire configuration file
type ApplicationConfig struct {
	Datasets          []Dataset              `json:"datasets"`
	DetectionMethods  DetectionMethodsParams `json:"detectionMethods"`
	GenCollectFilters CollectFilters         `json:"genCollectFilters"`
}

//Dataset provides the structure for each site configurations
//SiteCollectFilters field is an optional collection filter to be used for this site instead of the general filters
type Dataset struct {
	SiteId                  string          `json:"siteId"`
	TimeAgo                 string          `json:"timeAgo"`
	TimeStep                string          `json:"timeStep"`
	OutliersDetectionMethod string          `json:"outliersDetectionMethod"`
	MetricesList            []string        `json:"metricesList"`
	SiteCollectFilters      *CollectFilters `json:"siteCollectFilters"`
}

//DetectionMethodsParams provides the structure to store all detection methods parameters
type DetectionMethodsParams struct {
	ThreeSigmas ThreeSigmasParams `json:"3-sigmas"`
}

//ThreeSigmasParams provides the structure for the 3-sigmas detection method parameters
type ThreeSigmasParams struct {
	OutliersMultiplier       float64 `json:"outliersMultiplier"`
	StrongOutliersMultiplier float64 `json:"strongOutliersMultiplier"`
}

//CollectFilters provides the structure for collection filters
//AttributesFilterParams field is a map that points to the respective attributes parameters
type CollectFilters struct {
	MinVisitorsPerTimeStep int                     `json:"minVisitorsPerTimeStep"`
	AttributesFilterParams map[string]FilterParams `json:"attributesFilterParams"`
}

//FilterParams provides the structure for the attribute filter parameters
//Level field defines the maximum depth for that given attribute (0 for all)
//Top field defines the number of top sub-attributes in terms of samples count given attribute and level (0 for all)
type FilterParams struct {
	Level int `json:"level"`
	Top   int `json:"top"`
}

//ReadConfFile simply reads the configuration file
//It parses its contents in Json format and returns an ApplicationConfig structure
func ReadConfFile(confFile string) ApplicationConfig {

	//Opening and reading the configuration file, exiting the application if an error is detected
	byteValue, err := os.ReadFile(confFile)
	if err != nil {
		log.Fatalln(err.Error())
	}

	//Parsing the file content in Json format and returning the respective ApplicationConfig structure
	var appConf ApplicationConfig
	json.Unmarshal(byteValue, &appConf)

	return appConf
}
