package ApiGateway

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Appkube-awsx/awsx-common/authenticate"
	"github.com/Appkube-awsx/awsx-common/awsclient"
	"github.com/Appkube-awsx/awsx-common/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/spf13/cobra"
)

type ApiIntegrationLatencyResult struct {
	Value float64 `json:"Value"`
}

var AwsxApiIntegrationLatencyCmd = &cobra.Command{
	Use:   "api_integration_latency_panel",
	Short: "get integration latency metrics data",
	Long:  `command to get integration latency metrics data`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("running from child command")
		var authFlag, clientAuth, err = authenticate.AuthenticateCommand(cmd)
		if err != nil {
			log.Printf("Error during authentication: %v\n", err)
			err := cmd.Help()
			if err != nil {
				return
			}
			return
		}
		if authFlag {
			responseType, _ := cmd.PersistentFlags().GetString("responseType")
			jsonResp, cloudwatchMetricResp, err := GetApiIntegrationLatencyData(cmd, clientAuth, nil)
			if err != nil {
				log.Println("Error getting API integration latency data: ", err)
				return
			}
			if responseType == "frame" {
				fmt.Println(cloudwatchMetricResp)
			} else {
				fmt.Println(jsonResp)
			}
		}
	},
}

func GetApiIntegrationLatencyData(cmd *cobra.Command, clientAuth *model.Auth, cloudWatchClient *cloudwatch.CloudWatch) (string, map[string]float64, error) {
	ApiName := "dev-hrms"
	startTimeStr, _ := cmd.PersistentFlags().GetString("startTime")
	endTimeStr, _ := cmd.PersistentFlags().GetString("endTime")

	var startTime, endTime *time.Time

	// Parse start time if provided
	if startTimeStr != "" {
		parsedStartTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			log.Printf("Error parsing start time: %v", err)
			return "", nil, err
		}
		startTime = &parsedStartTime
	} else {
		defaultStartTime := time.Now().Add(-5 * time.Minute)
		startTime = &defaultStartTime
	}

	if endTimeStr != "" {
		parsedEndTime, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			log.Printf("Error parsing end time: %v", err)
			return "", nil, err
		}
		endTime = &parsedEndTime
	} else {
		defaultEndTime := time.Now()
		endTime = &defaultEndTime
	}

	// Debug prints
	log.Printf("StartTime: %v, EndTime: %v", startTime, endTime)

	cloudwatchMetricData := map[string]float64{}

	// Fetch raw data
	metricValue, err := GetApiLatencyMetricValue(clientAuth, startTime, endTime, ApiName, cloudWatchClient)
	if err != nil {
		log.Println("Error in getting latency metric value: ", err)
		return "", nil, err
	}
	cloudwatchMetricData["IntegrationLatency"] = metricValue

	// Debug prints
	log.Printf("Integration Latency Metric Value: %f", metricValue)

	jsonString, err := json.Marshal(MetricResult{Value: metricValue})
	if err != nil {
		log.Println("Error in marshalling json in string: ", err)
		return "", nil, err
	}

	return string(jsonString), cloudwatchMetricData, nil
}

func GetApiIntegrationLatencyMetricValue(clientAuth *model.Auth, startTime, endTime *time.Time, ApiName string, cloudWatchClient *cloudwatch.CloudWatch) (float64, error) {
	input := &cloudwatch.GetMetricDataInput{
		MetricDataQueries: []*cloudwatch.MetricDataQuery{
			{
				Id: aws.String("integration_latency"),
				MetricStat: &cloudwatch.MetricStat{
					Metric: &cloudwatch.Metric{
						Namespace:  aws.String("AWS/ApiGateway"),
						MetricName: aws.String("IntegrationLatency"),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("ApiName"),
								Value: aws.String(ApiName),
							},
						},
					},
					Period: aws.Int64(300),
					Stat:   aws.String("Average"),
				},
				ReturnData: aws.Bool(true),
			},
		},
		StartTime: startTime,
		EndTime:   endTime,
	}

	if cloudWatchClient == nil {
		cloudWatchClient = awsclient.GetClient(*clientAuth, awsclient.CLOUDWATCH).(*cloudwatch.CloudWatch)
	}

	result, err := cloudWatchClient.GetMetricData(input)
	if err != nil {
		return 0, err
	}

	if len(result.MetricDataResults) == 0 || len(result.MetricDataResults[0].Values) == 0 {
		return 0, fmt.Errorf("no data available for the specified time range")
	}

	// If there is only one value, return it
	if len(result.MetricDataResults[0].Values) == 1 {
		return aws.Float64Value(result.MetricDataResults[0].Values[0]), nil
	}

	// If there are multiple values, calculate the average
	var sum float64
	for _, v := range result.MetricDataResults[0].Values {
		sum += aws.Float64Value(v)
	}
	return sum / float64(len(result.MetricDataResults[0].Values)), nil
}

func init() {
	AwsxApiIntegrationLatencyCmd.PersistentFlags().String("elementId", "", "element id")
	AwsxApiIntegrationLatencyCmd.PersistentFlags().String("elementType", "", "element type")
	AwsxApiIntegrationLatencyCmd.PersistentFlags().String("query", "", "query")
	AwsxApiIntegrationLatencyCmd.PersistentFlags().String("cmdbApiUrl", "", "cmdb api")
	AwsxApiIntegrationLatencyCmd.PersistentFlags().String("vaultUrl", "", "vault end point")
	AwsxApiIntegrationLatencyCmd.PersistentFlags().String("vaultToken", "", "vault token")
	AwsxApiIntegrationLatencyCmd.PersistentFlags().String("zone", "", "aws region")
	AwsxApiIntegrationLatencyCmd.PersistentFlags().String("accessKey", "", "aws access key")
	AwsxApiIntegrationLatencyCmd.PersistentFlags().String("secretKey", "", "aws secret key")
	AwsxApiIntegrationLatencyCmd.PersistentFlags().String("crossAccountRoleArn", "", "aws cross account role arn")
	AwsxApiIntegrationLatencyCmd.PersistentFlags().String("externalId", "", "aws external id")
	AwsxApiIntegrationLatencyCmd.PersistentFlags().String("cloudWatchQueries", "", "aws cloudwatch metric queries")
	AwsxApiIntegrationLatencyCmd.PersistentFlags().String("instanceId", "", "instance id")
	AwsxApiIntegrationLatencyCmd.PersistentFlags().String("startTime", "", "start time")
	AwsxApiIntegrationLatencyCmd.PersistentFlags().String("endTime", "", "end time")
	AwsxApiIntegrationLatencyCmd.PersistentFlags().String("responseType", "", "response type. json/frame")
	AwsxApiIntegrationLatencyCmd.PersistentFlags().String("ApiName", "", "api name")
}
