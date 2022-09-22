# anomalies-detector
An exercise implementation of a e-Commerce Site Metrics Anomalies Detector

The exercise focuses on the Datasets Retrieval and Anomaly Detection modules of the given specification. 
![image](https://user-images.githubusercontent.com/97260490/191709248-7f80d55f-8f31-4bc4-b845-4488eaf6b3a4.png)

The application uses a JSON config file which is identified as an argument. That way, several different configurations can be setup and scheduled separately using Cron jobs or similar.

Since there is no access to the data repository in the exercise context, the Datasets Retrieval module is actually a data generator. Resulting datasets are random but they follow a normal distribution model. Hardcoded parameters allow to adjust the random distribution for each metric and also specifiy which attributes are returned.

The Anomaly Detection takes the collected Datasets and runs the detection algorithms specified on the configuration. For this exercise, only the 3-sigmas method was implemented but others can be easily added. The output is a report containing all warnings and alarms for each site in JSON format.

The alarms reports are meant to be used by the other application modules but on this exercise context, it simply stores the output in a JSON file. The same applies for the collected Datasets.

Although the exercise didn't include the Filtering and Reporting modules, it was extremely useful to have a mean to visualize the Datasets and respective alarms. So a basic reporting module was implemented using the charting library "github.com/wcharczuk/go-chart". After outputing the results to files, the application starts a web server allowing the user to select and download the charts. 

![Basket](https://user-images.githubusercontent.com/97260490/191707883-dd022750-9b1f-4119-96ed-e17768a4940f.png)

![Visits](https://user-images.githubusercontent.com/97260490/191717193-f61e59d5-e0b0-4fdc-a01d-0f0d9b52276d.png)
