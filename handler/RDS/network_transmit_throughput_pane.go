package RDS

import (
	"fmt"
	"log"

	"github.com/Appkube-awsx/awsx-common/authenticate"
	"github.com/Appkube-awsx/awsx-common/model"
	"github.com/Appkube-awsx/awsx-getelementdetails/comman-function"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/spf13/cobra"
)

// type NetworkTransmitThroughput struct {
// 	Timestamp time.Time
// 	Value     float64
// }

var AwsxRDSNetworkTransmitThroughputCmd = &cobra.Command{
	Use:   "network_transmit_throughput_panel",
	Short: "get network transmit throughput metrics data",
	Long:  `Command to get network transmit throughput metrics data`,

	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Running from child command")
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
			jsonResp, cloudwatchMetricResp, err := GetRDSNetworkTransmitThroughputPanel(cmd, clientAuth, nil)
			if err != nil {
				log.Println("Error getting network transmit throughput data: ", err)
				return
			}
			if responseType == "frame" {
				fmt.Println(cloudwatchMetricResp)
			} else {
				// Default case: print JSON
				fmt.Println(jsonResp)
			}
		}

	},
}

func GetRDSNetworkTransmitThroughputPanel(cmd *cobra.Command, clientAuth *model.Auth, cloudWatchClient *cloudwatch.CloudWatch) (string, map[string]*cloudwatch.GetMetricDataOutput, error) {

	elementType, _ := cmd.PersistentFlags().GetString("elementType")
	fmt.Println(elementType)
	instanceId, _ := cmd.PersistentFlags().GetString("instanceId")
	startTime, endTime, err := comman_function.ParseTimes(cmd)

	if err != nil {
		return "", nil, fmt.Errorf("error parsing time: %v", err)
	}
	instanceId, err = comman_function.GetCmdbData(cmd)

	if err != nil {
		return "", nil, fmt.Errorf("error getting instance ID: %v", err)

	}

	cloudwatchMetricData := map[string]*cloudwatch.GetMetricDataOutput{}

	rawData, err := comman_function.GetMetricData(clientAuth, instanceId, "AWS/RDS", "NetworkTransmitThroughput", startTime, endTime, "Sum", "DBInstanceIdentifier", cloudWatchClient)

	if err != nil {
		log.Println("Error in getting network transmit throughput data: ", err)
		return "", nil, err
	}
	cloudwatchMetricData["NetworkTransmitThroughput"] = rawData

	return "", cloudwatchMetricData, nil
}

// func processedRawNetworkTransmitThroughputData(result *cloudwatch.GetMetricDataOutput) []NetworkTransmitThroughput {
// 	var processedData []NetworkTransmitThroughput

// 	for i, timestamp := range result.MetricDataResults[0].Timestamps {
// 		value := *result.MetricDataResults[0].Values[i]
// 		processedData = append(processedData, NetworkTransmitThroughput{
// 			Timestamp: *timestamp,
// 			Value:     value,
// 		})
// 	}

// 	return processedData
// }

func init() {
	comman_function.InitAwsCmdFlags( AwsxRDSNetworkTransmitThroughputCmd)
}
