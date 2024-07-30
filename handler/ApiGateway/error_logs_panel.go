package ApiGateway

import (
	"fmt"

	"log"

	"github.com/Appkube-awsx/awsx-common/authenticate"
	"github.com/Appkube-awsx/awsx-common/model"
	"github.com/Appkube-awsx/awsx-getelementdetails/comman-function"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/spf13/cobra"
)

var AwsxApiErrorLogsCmd = &cobra.Command{

	Use:   "error_logs_panel",
	Short: "Get error logs metrics data",
	Long:  `Command to get error logs metrics data`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running error logs panel command")

		var authFlag bool
		var clientAuth *model.Auth
		var err error
		authFlag, clientAuth, err = authenticate.AuthenticateCommand(cmd)

		if err != nil {
			log.Printf("Error during authentication: %v\n", err)
			err := cmd.Help()
			if err != nil {
				return
			}
			return
		}
		if authFlag {
			panel, err := GetErrorLogsData(cmd, clientAuth, nil)
			if err != nil {
				return
			}
			fmt.Println(panel)
		}
	},
}

func GetErrorLogsData(cmd *cobra.Command, clientAuth *model.Auth, cloudWatchLogs *cloudwatchlogs.CloudWatchLogs) ([]*cloudwatchlogs.GetQueryResultsOutput, error) {
	logGroupName, _ := cmd.PersistentFlags().GetString("logGroupName")
	startTime, endTime, err := comman_function.ParseTimes(cmd)
	if err != nil {
		return nil, fmt.Errorf("error parsing time: %v", err)
	}
	logGroupName, err = comman_function.GetCmdbLogsData(cmd)
	if err != nil {
		return nil, fmt.Errorf("error getting instance ID: %v", err)
	}

	results, err := comman_function.GetLogsData(clientAuth, startTime, endTime, logGroupName, `fields @timestamp, eventType, eventSource, errorCode, errorMessage| filter eventSource = 'apigateway.amazonaws.com'| filter eventName ="GetMethod"| filter ispresent(responseElements) or ispresent(errorCode)| filter requestParameters.httpMethod != ""| stats count(errorMessage) as errorCode,count(eventTime) as ResponseTime by eventTime,errorMessage,requestParameters.httpMethod`, cloudWatchLogs)
	if err != nil {
		return nil, nil
	}
	processedResults := ProcessQueryResult(results)

	return processedResults, nil

}

func ProcessQueryResult(results []*cloudwatchlogs.GetQueryResultsOutput) []*cloudwatchlogs.GetQueryResultsOutput {
	processedResults := make([]*cloudwatchlogs.GetQueryResultsOutput, 0)

	for _, result := range results {
		if *result.Status == "Complete" {
			for _, resultField := range result.Results {
				for _, data := range resultField {
					if *data.Field == "eventTime" {

						log.Printf("eventTime: %s\n", *data)

					} else if *data.Field == "errorCode" {

						log.Printf("errorCode: %s\n", *data)

					} else if *data.Field == "errorMessage" {

						log.Printf("errorMessage: %s\n", *data)

					} else if *data.Field == "httpMethod" {

						log.Printf("httpMethod: %s\n", *data)
					}
				}
			}
			processedResults = append(processedResults, result)
		} else {
			log.Println("Query status is not complete.")
		}
	}

	return processedResults
}

func init() {
	comman_function.InitAwsCmdFlags(AwsxApiErrorLogsCmd)
}
