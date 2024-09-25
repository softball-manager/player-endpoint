package response

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
)

var (
	SuccessStatusCode             = 200
	BadRequestStatusCode          = 400
	InternalServerErrorStatusCode = 500
)

type SuccessfulResponse struct {
	PID    string `json:"pid"`
	Status string `json:"status"`
}

type BadRequestResponse struct {
	DeveloperText string `json:"developerText,omitempty"`
	Status        string `json:"status"`
}

type InternalServerErrorResponse struct {
	DeveloperText string `json:"developerText,omitempty"`
	Status        string `json:"status"`
}

func formatResponse(resp interface{}, statusCode int) events.APIGatewayProxyResponse {
	respJson, err := json.Marshal(resp)
	if err != nil {
		panic("unable to create response")
	}
	respStr := string(respJson)

	return events.APIGatewayProxyResponse{
		Body:       respStr,
		StatusCode: statusCode,
	}
}

func CreateSuccessfulResponse(pid string) events.APIGatewayProxyResponse {
	resp := &SuccessfulResponse{
		PID:    pid,
		Status: "Success",
	}

	return formatResponse(resp, SuccessStatusCode)
}

func CreateBadRequestResponse() events.APIGatewayProxyResponse {
	resp := &BadRequestResponse{
		Status: "Bad Request",
	}

	return formatResponse(resp, BadRequestStatusCode)
}

func CreateInternalServerErrorResponse() events.APIGatewayProxyResponse {
	resp := &InternalServerErrorResponse{
		Status: "Internal Server Error",
	}

	return formatResponse(resp, InternalServerErrorStatusCode)
}
