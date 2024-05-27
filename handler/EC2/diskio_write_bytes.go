package EC2

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Appkube-awsx/awsx-common/authenticate"
	"github.com/Appkube-awsx/awsx-common/awsclient"
	"github.com/Appkube-awsx/awsx-common/model"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/spf13/cobra"
)

var AwsxDiskIOWriteBytesCommmand = &cobra.Command{
	Use:   "diskio_write_bytes",
	Short: "get disk io write bytes metrics data",
	Long:  `command to get disk io write bytes metrics data`,
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
			jsonResp, resp, err := DiskIOWriteBytesData(cmd, clientAuth, nil, nil)
			if err != nil {
				log.Println("Error getting disk io write bytes data : ", err)
				return
			}
			if responseType == "json" {
				fmt.Println(jsonResp)
			} else {
				fmt.Println(resp)
			}
		}
	},
}

func DiskIOWriteBytesData(cmd *cobra.Command, clientAuth *model.Auth, ec2Client *ec2.EC2, cloudWatchClient *cloudwatch.CloudWatch) (string, string, error) {
	startTimeStr, _ := cmd.PersistentFlags().GetString("startTime")
	endTimeStr, _ := cmd.PersistentFlags().GetString("endTime")

	var startTime, endTime *time.Time

	// Parse start time if provided
	if startTimeStr != "" {
		parsedStartTime, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			log.Printf("Error parsing start time: %v", err)
			return "", "", err
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
			return "", "", err
		}
		endTime = &parsedEndTime
	} else {
		defaultEndTime := time.Now()
		endTime = &defaultEndTime
	}
	if ec2Client == nil {
		ec2Client = awsclient.GetClient(*clientAuth, awsclient.EC2_CLIENT).(*ec2.EC2)
	}
	ec2Input := ec2.DescribeInstancesInput{}
	instancesResult, err := ec2Client.DescribeInstances(&ec2Input)
	if err != nil {
		log.Printf("Error getting disk io write bytes data")
	}
	var instances []Ec2InstanceOutputData
	for _, reserv := range instancesResult.Reservations {
		for _, instance := range reserv.Instances {
			temp := Ec2InstanceOutputData{
				InstanceType: *instance.InstanceType,
				InstanceId:   *instance.InstanceId,
			}
			instances = append(instances, temp)
		}
	}
	// data := make(map[string]int)
	// data["full_concurrency"] = fullConcurrency
	fmt.Println("instances : ", instances)
	if cloudWatchClient == nil {
		cloudWatchClient = awsclient.GetClient(*clientAuth, awsclient.CLOUDWATCH).(*cloudwatch.CloudWatch)
	}
	var wg sync.WaitGroup

	ch := make(chan DiscIoWriteBytesRes)

	for _, instance := range instances {
		wg.Add(1)
		go getDiskIOWriteBytes(cloudWatchClient, instance, startTime, endTime, &wg, ch)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	var data []DiscIoWriteBytesRes
	for result := range ch {
		data = append(data, result)
	}

	jsonData, err := json.Marshal(data)
	fmt.Println("data : ", jsonData)
	if err != nil {
		log.Printf("error parsing data: %s", err)
		return "", "", err
	}
	return string(jsonData), string(jsonData), nil
}

type DiscIoWriteBytesRes struct {
	InstanceType string
	bytes        int64
}

func getDiskIOWriteBytes(cloudWatchClient *cloudwatch.CloudWatch, instance Ec2InstanceOutputData, startTime, endTime *time.Time, wg *sync.WaitGroup, ch chan<- DiscIoWriteBytesRes) {
	defer wg.Done()

	cwInput := cloudwatch.GetMetricDataInput{
		MetricDataQueries: []*cloudwatch.MetricDataQuery{
			{
				Id: aws.String("diskIoWriteBytes"),
				MetricStat: &cloudwatch.MetricStat{
					Metric: &cloudwatch.Metric{
						Namespace:  aws.String("CWAgent"),
						MetricName: aws.String("diskio_write_bytes"),
						Dimensions: []*cloudwatch.Dimension{
							{
								Name:  aws.String("InstanceId"),
								Value: aws.String(instance.InstanceId),
							},
						},
					},
					Period: aws.Int64(3600), // 1 hour in seconds
					Stat:   aws.String("Sum"),
				},
				ReturnData: aws.Bool(true),
			},
		},
		StartTime: aws.Time(time.Now().Add(-7 * 24 * time.Hour)), // 1 week ago
		EndTime:   aws.Time(time.Now()),
	}
	result, err := cloudWatchClient.GetMetricData(&cwInput)
	if err != nil {
		log.Printf("internal server error : %w", err)
	}
	values := result.MetricDataResults[0].Values
	var sum float64 = 0
	for _, item := range values {
		sum += *item
	}
	ch <- DiscIoWriteBytesRes{
		InstanceType: instance.InstanceType,
		bytes:        int64(sum),
	}
}
