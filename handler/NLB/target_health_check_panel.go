package NLB

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

type TargetHealthCheckData struct {
	HostCount map[string][]struct {
		Timestamp time.Time
		Value     float64
	} `json:"HostCount"`
}

var AwsxNLBTargetHealthChecksCmd = &cobra.Command{
	Use:   "nlb_target_health_checks_panel",
	Short: "Get NLB target health checks metrics data",
	Long:  `Command to get NLB target health checks metrics data`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running from child command..")
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
			jsonResp, cloudwatchMetricResp, err := GetNLBTargetHealthCheckPanel(cmd, clientAuth, nil)
			if err != nil {
				log.Println("Error getting NLB target health check data: ", err)
				return
			}
			if responseType == "frame" {
				fmt.Println(cloudwatchMetricResp)
			} else {
				// Default case. It prints JSON
				fmt.Println(jsonResp)
			}
		}

	},
}

func GetNLBTargetHealthCheckPanel(cmd *cobra.Command, clientAuth *model.Auth, cloudWatchClient *cloudwatch.CloudWatch) (string, map[string]*cloudwatch.GetMetricDataOutput, error) {
	nlbArn, _ := cmd.PersistentFlags().GetString("nlbArn")
	startTimeStr, _ := cmd.PersistentFlags().GetString("startTime")
	endTimeStr, _ := cmd.PersistentFlags().GetString("endTime")

	var startTime, endTime *time.Time

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

	log.Printf("StartTime: %v, EndTime: %v", startTime, endTime)

	cloudwatchMetricData := map[string]*cloudwatch.GetMetricDataOutput{}

	// Fetch raw data for successful host count
	successfulRawData, err := GetNLBTargetHealthCheckMetricData(clientAuth, nlbArn, startTime, endTime, "HealthyHostCount", "Average", cloudWatchClient)
	if err != nil {
		log.Println("Error in getting NLB successful target health check data: ", err)
		return "", nil, err
	}
	cloudwatchMetricData["Successful"] = successfulRawData

	// Fetch raw data for failed host count
	failedRawData, err := GetNLBTargetHealthCheckMetricData(clientAuth, nlbArn, startTime, endTime, "UnHealthyHostCount", "Average", cloudWatchClient)
	if err != nil {
		log.Println("Error in getting NLB failed target health check data: ", err)
		return "", nil, err
	}
	cloudwatchMetricData["Failed"] = failedRawData

	successfulHostCount := processNLBTargetHealthCheckRawData(successfulRawData)
	failedHostCount := processNLBTargetHealthCheckRawData(failedRawData)

	result := TargetHealthCheckData{
		HostCount: map[string][]struct {
			Timestamp time.Time
			Value     float64
		}{
			"SuccessfulHostCount": successfulHostCount,
			"FailedHostCount":     failedHostCount,
		},
	}

	jsonString, err := json.Marshal(result)
	if err != nil {
		log.Println("Error in marshalling json in string: ", err)
		return "", nil, err
	}

	return string(jsonString), cloudwatchMetricData, nil
}

func GetNLBTargetHealthCheckMetricData(clientAuth *model.Auth, nlbArn string, startTime, endTime *time.Time, metricName, statistic string, cloudWatchClient *cloudwatch.CloudWatch) (*cloudwatch.GetMetricDataOutput, error) {
	log.Printf("Getting metric data for NLB %s from %v to %v", nlbArn, startTime, endTime)

	input := &cloudwatch.GetMetricDataInput{
		EndTime:   endTime,
		StartTime: startTime,
		MetricDataQueries: []*cloudwatch.MetricDataQuery{
			{
				Id: aws.String("m1"),
				MetricStat: &cloudwatch.MetricStat{
					Metric: &cloudwatch.Metric{
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("LoadBalancer"),
								Value: aws.String("net/a0affec9643ca40c5a4e837eab2f07fb/f623f27b6210158f"),
							},
							{
								Name:  aws.String("TargetGroup"),
								Value: aws.String("targetgroup/k8s-istiosys-istioing-30129717de/b5e55c2955f8e65f"),
							},
						},
						MetricName: aws.String(metricName),
						Namespace:  aws.String("AWS/NetworkELB"),
					},
					Period: aws.Int64(60),
					Stat:   aws.String(statistic),
				},
			},
		},
	}
	if cloudWatchClient == nil {
		cloudWatchClient = awsclient.GetClient(*clientAuth, awsclient.CLOUDWATCH).(*cloudwatch.CloudWatch)
	}

	result, err := cloudWatchClient.GetMetricData(input)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func processNLBTargetHealthCheckRawData(result *cloudwatch.GetMetricDataOutput) []struct {
	Timestamp time.Time
	Value     float64
} {
	rawData := make([]struct {
		Timestamp time.Time
		Value     float64
	}, len(result.MetricDataResults[0].Timestamps))

	for i, timestamp := range result.MetricDataResults[0].Timestamps {
		rawData[i].Timestamp = *timestamp
		rawData[i].Value = *result.MetricDataResults[0].Values[i]
	}

	return rawData
}

func init() {
	AwsxNLBTargetHealthChecksCmd.PersistentFlags().String("nlbArn", "", "NLB ARN")
	AwsxNLBTargetHealthChecksCmd.PersistentFlags().String("startTime", "", "start time")
	AwsxNLBTargetHealthChecksCmd.PersistentFlags().String("endTime", "", "end time")
	AwsxNLBTargetHealthChecksCmd.PersistentFlags().String("responseType", "", "response type. json/frame")
}
