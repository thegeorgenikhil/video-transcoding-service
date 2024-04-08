package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/ecs"
)

type EventDetail struct {
	Timestamp       time.Time `json:"timestamp"`
	Version         string    `json:"version"`
	Bucket          Bucket    `json:"bucket"`
	Object          Object    `json:"object"`
	RequestID       string    `json:"request-id"`
	Requester       string    `json:"requester"`
	SourceIPAddress string    `json:"source-ip-address"`
	Reason          string    `json:"reason"`
}

type Bucket struct {
	Name string `json:"name"`
}

type Object struct {
	Key       string `json:"key"`
	Size      int    `json:"size"`
	ETag      string `json:"etag"`
	Sequencer string `json:"sequencer"`
}

type App struct {
	ecsCl    *ecs.ECS
	dynamoCl *dynamodb.DynamoDB
}

func main() {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("ap-south-1"),
	})
	if err != nil {
		fmt.Println("Error creating session", err)
		return
	}

	ecsClient := ecs.New(sess)
	dynamoClient := dynamodb.New(sess)

	app := App{
		ecsCl:    ecsClient,
		dynamoCl: dynamoClient,
	}

	lambda.Start(app.HandleRequest)
}

func (app *App) HandleRequest(event events.EventBridgeEvent) {
	var detail EventDetail
	err := json.Unmarshal(event.Detail, &detail)
	if err != nil {
		fmt.Println("Error unmarshalling event detail", err)
		return
	}

	_, err = app.dynamoCl.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String("Videos"),
		Item: map[string]*dynamodb.AttributeValue{
			"Key": {
				S: aws.String(detail.Object.Key),
			},
			"UploadedAt": {
				S: aws.String(time.Now().Format(time.RFC3339)),
			},
			"Status": {
				S: aws.String("uploaded"),
			},
		},
	})
	if err != nil {
		log.Fatal("Error putting item in DynamoDB", err)
		return
	}

	_, err = app.ecsCl.RunTask(&ecs.RunTaskInput{
		Cluster:        aws.String(""), //TODO: get from environment
		TaskDefinition: aws.String(""), //TODO: get from environment
		LaunchType:     aws.String("FARGATE"),
		Count:          aws.Int64(1),
		NetworkConfiguration: &ecs.NetworkConfiguration{
			AwsvpcConfiguration: &ecs.AwsVpcConfiguration{
				AssignPublicIp: aws.String("ENABLED"),
				Subnets: []*string{
					aws.String(""), //TODO: get from environment
					aws.String(""), //TODO: get from environment
					aws.String(""), //TODO: get from environment
				},
			},
		},
		Overrides: &ecs.TaskOverride{
			ContainerOverrides: []*ecs.ContainerOverride{
				{
					Name: aws.String("video-transcoding-image"),
					Environment: []*ecs.KeyValuePair{
						{
							Name:  aws.String("TEMPORARY_BUCKET_NAME"),
							Value: aws.String(detail.Bucket.Name),
						},
						{
							Name:  aws.String("OUTPUT_BUCKET_NAME"),
							Value: aws.String(""), //TODO: get from environment
						},
						{
							Name:  aws.String("BUCKET_REGION"),
							Value: aws.String("ap-south-1"),
						},
						{
							Name:  aws.String("OBJECT_KEY"),
							Value: aws.String(detail.Object.Key),
						},
					},
				},
			},
		},
	})

	if err != nil {
		fmt.Println("Error running task", err)
		return
	}

	fmt.Println("Received event for object", detail.Object.Key)
}
