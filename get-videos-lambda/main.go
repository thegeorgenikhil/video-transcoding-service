package main

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

const (
	REGION      = "ap-south-1"
	BUCKET_NAME = "" // TODO: get from enviroment

	EXPIRY_IN_MINUTES = 60 * time.Minute
)

type RequestBody struct {
	AccessToken string `json:"access_token"`
}

type Response struct {
	Videos []Video `json:"videos"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

type App struct {
	token    string
	dynamoCl *dynamodb.DynamoDB
}

type Video struct {
	Key             string            `json:"key" dynamodbav:"Key"`
	Status          string            `json:"status" dynamodbav:"Status"`
	TranscodingTime string            `json:"transcoding_time" dynamodbav:"TranscodingTime"`
	UploadedAt      string            `json:"uploaded_at" dynamodbav:"UploadedAt"`
}

func main() {
	token := os.Getenv("GET_VIDEOS_ACCESS_TOKEN")
	if token == "" {
		log.Fatalf("GET_VIDEOS_ACCESS_TOKEN is not set")
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(REGION),
	})
	if err != nil {
		log.Fatalf("failed to create AWS session, %v", err)
	}

	dynamoClient := dynamodb.New(sess)

	app := App{token: token, dynamoCl: dynamoClient}

	lambda.Start(app.HandleRequest)
}

func (app *App) HandleRequest(request events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	var reqBody RequestBody
	err := json.Unmarshal([]byte(request.Body), &reqBody)
	if err != nil {
		log.Printf("failed to unmarshal request body, %v\n", err)
		return nil, err
	}

	if reqBody.AccessToken == "" {
		errResp, err := generateErrorResponse("access token is missing", 400)
		if err != nil {
			log.Printf("failed to generate error response, %v\n", err)
			return nil, err
		}

		return errResp, nil
	}

	if reqBody.AccessToken != app.token {
		errResp, err := generateErrorResponse("invalid access token", 401)
		if err != nil {
			log.Printf("failed to generate error response, %v\n", err)
			return nil, err
		}

		return errResp, nil
	}

	input := &dynamodb.ScanInput{
		TableName: aws.String("Videos"),
	}

	result, err := app.dynamoCl.Scan(input)
	if err != nil {
		log.Println("failed to scan videos, ", err)
		return nil, err
	}

	videos := []Video{}
	for _, item := range result.Items {
		video := Video{}
		err = dynamodbattribute.UnmarshalMap(item, &video)
		if err != nil {
			log.Printf("failed to unmarshal video, %v\n", err)
			return nil, err
		}

		videos = append(videos, video)
	}

	resp, err := json.Marshal(Response{Videos: videos})
	if err != nil {
		log.Printf("failed to marshal response, %v\n", err)
		return nil, err
	}

	return &events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(resp),
	}, nil
}

func generateErrorResponse(msg string, status int) (*events.APIGatewayProxyResponse, error) {
	errMsg := ErrorResponse{Message: msg}
	body, err := json.Marshal(errMsg)
	if err != nil {
		log.Printf("failed to marshal error response, %v\n", err)
		return nil, err
	}

	return &events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       string(body),
	}, nil
}
