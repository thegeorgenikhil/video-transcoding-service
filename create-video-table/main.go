package main

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

const (
	REGION = "ap-south-1"
)

type VideoItem struct {
	Key        string
	UploadedAt string
}

func main() {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(REGION),
	})
	if err != nil {
		log.Fatalf("failed to create AWS session, %v", err)
	}

	dynamoClient := dynamodb.New(sess)

	_, err = dynamoClient.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String("Videos"),
	})
	if err != nil {
		_, err = dynamoClient.CreateTable(&dynamodb.CreateTableInput{
			TableName: aws.String("Videos"),
			KeySchema: []*dynamodb.KeySchemaElement{
				{
					AttributeName: aws.String("Key"),
					KeyType:       aws.String("HASH"),
				},
			},
			AttributeDefinitions: []*dynamodb.AttributeDefinition{
				{
					AttributeName: aws.String("Key"),
					AttributeType: aws.String("S"),
				},
			},
			ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
				ReadCapacityUnits:  aws.Int64(10),
				WriteCapacityUnits: aws.Int64(10),
			},
		})
		if err != nil {
			log.Fatalf("failed to create table, %v", err)
		}
		log.Println("table created successfully")
	} else {
		log.Println("table already exists")
	}
}
