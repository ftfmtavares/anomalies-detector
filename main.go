package main

import (
	"errors"
	"flag"
	"log"
	"os"

	"github.com/ftfmtavares/anomalies-detector/analyser"
	"github.com/ftfmtavares/anomalies-detector/collector"
	"github.com/ftfmtavares/anomalies-detector/config"
	"github.com/ftfmtavares/anomalies-detector/reporting"
	"github.com/ftfmtavares/anomalies-detector/utils"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ldate + log.Ltime + log.Lmicroseconds)

	//Defining CLI arguments using the flag package
	//Default values are local files with standard names and no overwrite option
	confFile := flag.String("conf-file", "config.json", "Configuration file name")
	dataFile := flag.String("data-file", "data.json", "Collected Data file name")
	reportFile := flag.String("report-file", "report.json", "Outliers Report file name")
	overwrite := flag.Bool("overwrite", false, "Overwrite existing files")
	flag.Parse()

	//Validating the arguments values
	if err := validateInputFile(*confFile); err != nil {
		log.Fatalf("conf-file \"%s\" - %s\n\n", *confFile, err.Error())
	}
	if err := validateOutputFile(*dataFile, *overwrite); err != nil {
		log.Fatalf("data-file \"%s\" - %s\n\n", *dataFile, err.Error())
		return
	}
	if err := validateOutputFile(*reportFile, *overwrite); err != nil {
		log.Fatalf("report-file \"%s\" - %s\n\n", *reportFile, err.Error())
		return
	}

	//Reading configurations from the config file
	log.Printf("Using configuration file \"%s\"\n", *confFile)
	config := config.ReadConfFile(*confFile)
	log.Println("Configuration Read:")
	utils.PrintJsonStruct(config)

	sitesData := []collector.SiteData{}
	reports := []analyser.OutlierReport{}

	//Looping all sites from the configuration file
	for _, dataSet := range config.Datasets {

		//Using general collection filters if none defined for the specific site
		if dataSet.SiteCollectFilters == nil {
			dataSet.SiteCollectFilters = &config.GenCollectFilters
		}

		//Reading and adding data to the slice
		siteData := collector.GetData(dataSet)
		sitesData = append(sitesData, siteData)

		//Analysing and adding report to the slice
		report := analyser.GetResults(siteData, dataSet, config.DetectionMethods)
		reports = append(reports, report)
	}

	//Exporting both data and reports on given files
	utils.WriteJsonStruct(sitesData, *dataFile)
	utils.WriteJsonStruct(reports, *reportFile)

	//Starting an web server with visual information of collected data and detected alarms
	//For the exercise results visual presentation only, it should be replaced by the final report module with slack integration
	log.Println("Generated Report on http://localhost:8080/report")
	reporting.GenerateReport(sitesData, reports, 8080)
}

//validateInputFile checks if a given file name is valid to be read
//It returns an error if file name is empty or invalid, if file does not exist or if it's a directory
func validateInputFile(inputFile string) error {
	if inputFile == "" {
		return errors.New("missing parameter")
	}
	if fileInfo, err := os.Stat(inputFile); err != nil || fileInfo.IsDir() {
		if err != nil && os.IsNotExist(err) {
			return errors.New("file does not exist")
		} else if fileInfo.IsDir() {
			return errors.New("file is a directory")
		} else {
			return errors.New("invalid file name")
		}
	}

	return nil
}

//validateOutputFile checks if a given file name is valid to be writen with overwrite option or not
//It returns an error if file name is empty or invalid, if it's a directory or it simply fails to create
//An empty file is actually created at this stage in order to test any possible creation errors (lack of permissions for instance)
func validateOutputFile(outputFile string, overwrite bool) error {
	if outputFile == "" {
		return errors.New("missing parameter")
	}
	if fileInfo, err := os.Stat(outputFile); err == nil || !os.IsNotExist(err) {
		if err != nil && !os.IsNotExist(err) {
			return err
		} else if fileInfo.IsDir() {
			return errors.New("file is a directory")
		} else if !overwrite {
			return errors.New("file already exists")
		}
	}
	fData, err := os.Create(outputFile)
	if err != nil {
		return err
	}
	defer fData.Close()

	return nil
}
