package request

import (
	"encoding/json"

	"github.com/aws/aws-lambda-go/events"
	"github.com/go-playground/validator/v10"
)

type Request struct {
	Name      string   `json:"name" validate:"required"`
	Positions []string `json:"positions"`
}

func ValidateRequest(request events.APIGatewayProxyRequest) (*Request, error) {
	var validRequest Request

	err := json.Unmarshal([]byte(request.Body), &validRequest)
	if err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(&validRequest); err != nil {
		return nil, err
	}

	return &validRequest, nil
}
