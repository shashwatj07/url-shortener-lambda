package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/teris-io/shortid"
)

const (
	LinksTableName = "UrlShortenerLinks"
	Region         = "ap-south-1"
)

type Request struct {
	URL   string `json:"url"`
	Alias string `json:"alias"`
	Validity  int    `json:"validity"`
}

type Response struct {
	ShortURL string `json:"short_url"`
}

type Link struct {
	ShortURL string `json:"short_url"`
	LongURL  string `json:"long_url"`
	ExpDate  int    `json:"exp_date"`
}

// Start DynamoDB session
var sess, sess_err = session.NewSession(&aws.Config{
	Region: aws.String(Region),
})

var svc = dynamodb.New(sess)

func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Setup CORS header
	resp := events.APIGatewayProxyResponse{
		Headers: make(map[string]string),
	}
	resp.Headers["Access-Control-Allow-Origin"] = "*"
	// Parse request body
	rb := Request{}
	if err := json.Unmarshal([]byte(request.Body), &rb); err != nil {
		return resp, err
	}
	// If session not started correctly, try again
	if sess_err != nil {
		sess, sess_err := session.NewSession(&aws.Config{
			Region: aws.String(Region),
		})
		if sess_err != nil {
			return resp, sess_err
		}
		svc = dynamodb.New(sess)
	}
	link := &Link{
		ShortURL: rb.Alias,
		LongURL:  rb.URL,
		ExpDate:  int(time.Now().AddDate(0, 0, rb.Validity).Unix()),
	}
	if link.ShortURL == "" {
		// Generate short url
		link.ShortURL = shortid.MustGenerate()

	}
	// Because "shorten" endpoint is reserved
	for link.ShortURL == "shorten" {
		link.ShortURL = shortid.MustGenerate()
	}
	// Marshal link to attribute value map
	av, err := dynamodbattribute.MarshalMap(link)
	if err != nil {
		return resp, err
	}
	// Put link
	input := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String(LinksTableName),
	}
	if _, err = svc.PutItem(input); err != nil {
		return resp, err
	}
	// Return short url
	response, err := json.Marshal(Response{ShortURL: link.ShortURL})
	if err != nil {
		return resp, err
	}
	resp.StatusCode = http.StatusOK
	resp.Body = string(response)

	return resp, nil
}

func main() {
	lambda.Start(Handler)
}
