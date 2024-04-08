package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var (
	__temporaryBucketName = os.Getenv("TEMPORARY_BUCKET_NAME")
	__outputBucketName    = os.Getenv("OUTPUT_BUCKET_NAME")
	__bucketRegion        = os.Getenv("BUCKET_REGION")
	__objectKey           = os.Getenv("OBJECT_KEY")

	transcodingFormats = TranscodingFormatMap{
		"144p":  "256:144",
		"240p":  "426:240",
		"360p":  "640:360",
		"480p":  "854:480",
		"720p":  "1280:720",
		"1080p": "1920:1080",
	}

	wg sync.WaitGroup

	transcodedVideoInfoMap = TranscodedVideoInfo{
		infoMap: make(map[string]string),
	}
)

type TranscodingFormatMap map[string]string

type TranscodedVideoInfo struct {
	infoMap map[string]string
	sync.Mutex
}

func (t *TranscodedVideoInfo) AddInfo(resolution string, url string) {
	t.Lock()
	defer t.Unlock()
	t.infoMap[resolution] = url
}

func main() {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(__bucketRegion),
	})

	if err != nil {
		log.Fatalf("failed to create AWS session, %v", err)
	}

	s3Downloader := s3manager.NewDownloader(sess)
	dynamoClient := dynamodb.New(sess)

	file, err := os.Create(__objectKey)
	if err != nil {
		log.Fatalf("failed to create file %q, %v", __objectKey, err)
	}

	defer file.Close()

	_, err = dynamoClient.UpdateItem(&dynamodb.UpdateItemInput{
		TableName: aws.String("Videos"),
		ExpressionAttributeNames: map[string]*string{
			"#S": aws.String("Status"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":s": {
				S: aws.String("processing"),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"Key": {
				S: aws.String(__objectKey),
			},
		},
		UpdateExpression: aws.String("SET #S = :s"),
	})
	if err != nil {
		log.Fatalf("failed to update item in DynamoDB, %v", err)
	}

	// STEP 1: Download the video from S3
	_, err = s3Downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(__temporaryBucketName),
			Key:    aws.String(__objectKey),
		})
	if err != nil {
		log.Fatalf("failed to download file, %v", err)
	}

	// STEP 2: Transcode the video to all resolutions
	videoFilePath := "./" + file.Name()

	startTime := time.Now()
	for r := range transcodingFormats {
		outputName := getFormattedOutputName(file.Name(), r)

		wg.Add(1)
		go transcodeVideo(videoFilePath, outputName, r, &wg, &transcodedVideoInfoMap)
	}

	wg.Wait()

	totalTime := time.Since(startTime)

	// STEP 3: Upload the transcoded videos to S3
	svc := s3.New(sess)

	for r, url := range transcodedVideoInfoMap.infoMap {
		file, err := os.Open(url)
		if err != nil {
			log.Fatal("Error opening file:", err)
		}
		defer file.Close()

		key := getFormattedOutputName(__objectKey, r)

		// Read the contents of the file into a buffer
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, file); err != nil {
			log.Fatal("Error reading file:", err)
		}

		// This uploads the contents of the buffer to S3
		_, err = svc.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(__outputBucketName),
			Key:    aws.String(key),
			Body:   bytes.NewReader(buf.Bytes()),
		})
		if err != nil {
			log.Fatal("Error uploading data to S3:", err)
		}

		fmt.Println("File uploaded successfully!!! ", file.Name())
	}

	_, err = dynamoClient.UpdateItem(&dynamodb.UpdateItemInput{
		TableName: aws.String("Videos"),
		ExpressionAttributeNames: map[string]*string{
			"#S": aws.String("Status"),
			"#T": aws.String("TranscodingTime"),
			"#F": aws.String("TranscodedFiles"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":s": {
				S: aws.String("completed"),
			},
			":t": {
				S: aws.String(fmt.Sprintf("%f", totalTime.Seconds())),
			},
			":f": {
				M: map[string]*dynamodb.AttributeValue{
					"144p":  {S: aws.String(getFormattedOutputName(__objectKey, "144p"))},
					"240p":  {S: aws.String(getFormattedOutputName(__objectKey, "240p"))},
					"360p":  {S: aws.String(getFormattedOutputName(__objectKey, "360p"))},
					"480p":  {S: aws.String(getFormattedOutputName(__objectKey, "480p"))},
					"720p":  {S: aws.String(getFormattedOutputName(__objectKey, "720p"))},
					"1080p": {S: aws.String(getFormattedOutputName(__objectKey, "1080p"))},
				},
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"Key": {
				S: aws.String(__objectKey),
			},
		},
		UpdateExpression: aws.String("SET #S = :s, #T = :t, #F = :f"),
	})
	if err != nil {
		log.Fatalf("failed to update item in DynamoDB, %v", err)
	}
}

func getFormattedOutputName(videoFileName string, resolution string) string {
	return strings.Split(videoFileName, ".")[0] + "_" + resolution + "." + strings.Split(videoFileName, ".")[1]
}

func transcodeVideo(filePath string, outputFileName string, resolution string, wg *sync.WaitGroup, transcodedVideoInfoMap *TranscodedVideoInfo) {
	defer wg.Done()
	outputFilePath := "./out/" + outputFileName

	scale := transcodingFormats[resolution]

	fmt.Printf("ffmpeg -i %s -vf scale=%s -acodec copy -c:a copy %s\n", filePath, scale, outputFilePath)

	cmd := exec.Command("ffmpeg", "-i", filePath, "-vf", "scale="+scale, "-acodec", "copy", "-c:a", "copy", outputFilePath)
	stderr, _ := cmd.StderrPipe()
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}

	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

	transcodedVideoInfoMap.AddInfo(resolution, outputFilePath)
}
