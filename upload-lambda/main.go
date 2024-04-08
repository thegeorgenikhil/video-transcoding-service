package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
)

const (
	REGION      = "ap-south-1"
	BUCKET_NAME = "video-transcoding-temp"

	EXPIRY_IN_MINUTES = 60 * time.Minute
)

type RequestBody struct {
	AccessToken string `json:"access_token"`
	FileName    string `json:"file_name"`
}

type Response struct {
	Key          string `json:"key"`
	PreSignedURL string `json:"upload_url"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

type App struct {
	Token    string
	S3       *s3.S3
}

func main() {
	token := os.Getenv("UPLOAD_LAMBDA_ACCESS_TOKEN")
	if token == "" {
		log.Fatalf("UPLOAD_LAMBDA_ACCESS_TOKEN is not set")
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(REGION),
	})
	if err != nil {
		log.Fatalf("failed to create AWS session, %v", err)
	}

	s3 := s3.New(sess)

	app := App{S3: s3, Token: token}

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

	if reqBody.AccessToken != app.Token {
		errResp, err := generateErrorResponse("invalid access token", 401)
		if err != nil {
			log.Printf("failed to generate error response, %v\n", err)
			return nil, err
		}

		return errResp, nil
	}

	if reqBody.FileName == "" {
		errResp, err := generateErrorResponse("file name is missing", 400)
		if err != nil {
			log.Printf("failed to generate error response, %v\n", err)
			return nil, err
		}

		return errResp, nil
	}

	info := strings.Split(reqBody.FileName, ".")
	fileName := info[0]
	exts := info[1]

	uuid, err := uuid.NewV7()
	if err != nil {
		log.Printf("failed to generate UUID, %v\n", err)
		return nil, err
	}

	key := fmt.Sprintf("%s-%s.%s", fileName, uuid, exts)

	url, err := app.GetPresignedUploadURL(key)
	if err != nil {
		log.Printf("failed to get presigned URL, %v\n", err)
		return nil, err
	}

	resp, err := json.Marshal(Response{Key: key, PreSignedURL: url})
	if err != nil {
		log.Printf("failed to marshal response, %v\n", err)
		return nil, err
	}

	return &events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(resp),
	}, nil
}

func (app *App) GetPresignedUploadURL(key string) (string, error) {
	req, _ := app.S3.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(BUCKET_NAME),
		Key:    aws.String(key),
	})

	url, err := req.Presign(EXPIRY_IN_MINUTES)

	if err != nil {
		return "", err
	}

	return url, nil
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
