package reporting

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/ftfmtavares/anomalies-detector/analyser"
	"github.com/ftfmtavares/anomalies-detector/collector"
	"github.com/ftfmtavares/anomalies-detector/utils"

	"github.com/gorilla/mux"
	"github.com/wcharczuk/go-chart/v2"
	"github.com/wcharczuk/go-chart/v2/drawing"
)

//GenerateReport takes all collected data and alarm reports and starts an web server from which different graphs can be downloaded
func GenerateReport(sitesData []collector.SiteData, outlierReports []analyser.OutlierReport, port int) {

	//writeIndex implements an HTTP response returning a simple HTML bullet list with links to all available sites, metrics and main attributes
	writeIndex := func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		res.Write([]byte("<!DOCTYPE html>\n"))
		res.Write([]byte("<title>Anomalies Report</title>\n"))
		for _, siteData := range sitesData {
			res.Write([]byte(fmt.Sprintf("<h2>%s</h2>\n", siteData.SiteId)))
			res.Write([]byte("<ul>\n"))
			for _, metricData := range siteData.Metrics {
				res.Write([]byte(fmt.Sprintf("<li><a href=\"/report/%s/%s\">%s</a></li>\n", siteData.SiteId, metricData.Metric, metricData.Metric)))
				res.Write([]byte("<ul>\n"))
				lastAttribute := ""
				for _, attribute := range metricData.Attributes {
					parts := strings.Split(attribute, ">")
					if parts[0] != lastAttribute {
						lastAttribute = parts[0]
						res.Write([]byte(fmt.Sprintf("<li><a href=\"/report/%s/%s?attribute=%s\">%s</a></li>\n", siteData.SiteId, metricData.Metric, strings.ToLower(lastAttribute), lastAttribute)))
					}
				}
				res.Write([]byte("</ul>\n"))
			}
			res.Write([]byte("</ul>\n"))
			res.Write([]byte("<hr />\n"))
		}
	}

	//writeIndex implements an HTTP response returning PNG images containing graphs with collected data and alarms annotations
	drawChart := func(res http.ResponseWriter, req *http.Request) {

		//It takes the site id and metric from the url address, as well as attributes from query strings, to generate the graph on demand
		siteUrl := mux.Vars(req)["siteid"]
		metricUrl := mux.Vars(req)["metric"]
		attributesUrl := req.URL.Query()["attribute"]

		//If "all" or no attribute has been given in query strings attributes, all attribute/sub-value combinations will be shown
		allAttributes := false
		if len(attributesUrl) == 0 {
			allAttributes = true
		} else {
			for _, attr := range attributesUrl {
				if strings.ToLower(attr) == "all" {
					allAttributes = true
					break
				}
			}
		}

		//Looks for the respective metric data
		chosenMetric := collector.MetricData{}
	OuterLoop:
		for _, siteData := range sitesData {
			if siteData.SiteId == siteUrl {
				for _, metric := range siteData.Metrics {
					if metric.Metric == metricUrl {
						chosenMetric = metric
						break OuterLoop
					}
				}
			}
		}

		//If an unknown site and metric was given, an HTTP not found error is returned, otherwise the respective graph is generated
		if chosenMetric.Metric == "" {
			res.WriteHeader(http.StatusNotFound)
			res.Write([]byte("404 page not found\n"))
		} else {
			graph := chart.Chart{
				Title:  fmt.Sprintf("%s - %s", siteUrl, metricUrl),
				Width:  1366,
				Height: 768,
				Background: chart.Style{
					Padding: chart.Box{
						Top:  30,
						Left: 160,
					},
				},
				XAxis: chart.XAxis{
					Name: "Time",
				},
				YAxis: chart.YAxis{
					Name: chosenMetric.Unit,
				},
				Series: []chart.Series{},
			}

			max := 0.0
			shownAttributes := map[string]bool{}
			//Looping through the available attribute/sub-value combinations in the selected metric data
			for _, attribute := range chosenMetric.Attributes {

				//Checking if the attribute/sub-value combination is to be shown and stores in a map for future use
				if !allAttributes {
					for _, attr := range attributesUrl {
						if strings.HasPrefix(strings.ToLower(attribute), strings.ToLower(attr)) {
							shownAttributes[attribute] = true
							break
						}
					}
				}

				//Adding the data series in the graph if the attribute/sub-value combination is to be shown
				if allAttributes || shownAttributes[attribute] {
					newSeries := chart.TimeSeries{
						Name:    attribute,
						XValues: make([]time.Time, len(chosenMetric.AttributeData[attribute])),
						YValues: make([]float64, len(chosenMetric.AttributeData[attribute])),
					}
					for i, timeStepData := range chosenMetric.AttributeData[attribute] {
						newSeries.XValues[i] = timeStepData.DateStart
						newSeries.YValues[i] = timeStepData.Value
						if max < timeStepData.Value {
							max = timeStepData.Value
						}
					}
					graph.Series = append(graph.Series, newSeries)
				}
			}

			//Looping through all alarms, checking if they belong to the shown metric and attributes, and adding them as annotations in the graph
			alarmsMarkup := map[string]chart.AnnotationSeries{}
			for _, outlierReport := range outlierReports {
				if outlierReport.SiteId == siteUrl {
					for _, alarm := range outlierReport.Result.Alarms {
						if alarm.Metric == metricUrl && (allAttributes || shownAttributes[alarm.Attribute]) {
							if _, present := alarmsMarkup[strings.Join([]string{alarm.OutlierPeriodStart.String(), alarm.OutlierPeriodEnd.String()}, "")]; !present {
								xOffset, _ := utils.StrToDuration(outlierReport.TimeStep)
								xOffset = -1 * xOffset / 2

								newAlarmShade := chart.TimeSeries{
									Name: "",
									Style: chart.Style{
										StrokeWidth: 0,
										StrokeColor: drawing.Color{R: 255, G: 0, B: 0, A: 0},
										DotColor:    drawing.Color{R: 255, G: 0, B: 0, A: 0},
										DotWidth:    0,
										FillColor:   drawing.Color{R: 255, G: 0, B: 0, A: 40},
									},
									XValues: []time.Time{alarm.OutlierPeriodStart.Add(xOffset), alarm.OutlierPeriodEnd.Add(xOffset)},
									YValues: []float64{max, max},
								}
								graph.Series = append(graph.Series, newAlarmShade)

								xOffset2, _ := utils.StrToDuration(outlierReport.TimeAgo)
								xOffset = xOffset - 1*xOffset2/100

								label := alarm.Attribute
								parts := strings.Split(label, ">")
								if len(parts) > 1 {
									parts = parts[1:]
									label = strings.Join(parts, ">")
								}

								newAlarmAnnotation := chart.AnnotationSeries{
									Style: chart.Style{
										DotColor:            drawing.Color{R: 255, G: 0, B: 0, A: 0},
										FillColor:           drawing.Color{R: 255, G: 0, B: 0, A: 0},
										StrokeColor:         drawing.Color{R: 255, G: 0, B: 0, A: 0},
										FontColor:           drawing.Color{R: 255, G: 0, B: 0, A: 255},
										FontSize:            8,
										TextRotationDegrees: 90,
									},
									Annotations: []chart.Value2{{Label: label, XValue: float64(alarm.OutlierPeriodEnd.Add(xOffset).UnixNano()), YValue: max}},
								}
								graph.Series = append(graph.Series, newAlarmAnnotation)

								alarmsMarkup[strings.Join([]string{alarm.OutlierPeriodStart.String(), alarm.OutlierPeriodEnd.String()}, "")] = newAlarmAnnotation
							} else {
								newLabel := alarm.Attribute
								parts := strings.Split(newLabel, ">")
								if len(parts) > 1 {
									parts = parts[1:]
									newLabel = strings.Join(parts, ">")
								}

								parts = strings.Split(alarmsMarkup[strings.Join([]string{alarm.OutlierPeriodStart.String(), alarm.OutlierPeriodEnd.String()}, "")].Annotations[0].Label, "+")
								valid := true
								for _, part := range parts {
									if part == "Total" || strings.HasPrefix(newLabel, part) {
										valid = false
										break
									}
								}

								if valid {
									alarmsMarkup[strings.Join([]string{alarm.OutlierPeriodStart.String(), alarm.OutlierPeriodEnd.String()}, "")].Annotations[0].Label = fmt.Sprintf("%s+%s", alarmsMarkup[strings.Join([]string{alarm.OutlierPeriodStart.String(), alarm.OutlierPeriodEnd.String()}, "")].Annotations[0].Label, newLabel)
								}
							}
						}
					}
					break
				}
			}

			graph.YAxis.Range = &chart.ContinuousRange{
				Min: 0.0,
				Max: max * 1.2,
			}

			graph.Elements = []chart.Renderable{
				chart.LegendLeft(&graph),
			}

			res.Header().Set("Content-Type", "image/png")
			graph.Render(chart.PNG, res)
		}
	}

	//Registers both index and chart functions as handles and start the web server
	router := mux.NewRouter()
	router.PathPrefix("/report").Methods(http.MethodOptions, http.MethodGet).Subrouter().HandleFunc("", writeIndex)
	router.PathPrefix("/report/{siteid}/{metric}").Methods(http.MethodOptions, http.MethodGet).Subrouter().HandleFunc("", drawChart)
	srv := http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf(":%d", port),
		WriteTimeout: 10 * time.Second,
		ReadTimeout:  10 * time.Second,
	}
	srv.ListenAndServe()
}
