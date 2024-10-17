package request

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"

	"github.com/aws/aws-lambda-go/events"
	"github.com/go-playground/validator/v10"
	"github.com/softball-manager/common/pkg/dynamo"
)

type CreatePlayerRequest struct {
	Name      string   `json:"name" validate:"required"`
	Positions []string `json:"positions"`
}

var (
	validPidRegex = fmt.Sprintf(`^%s[a-zA-Z0-9-]+$`, dynamo.PlayerIDPrefix)
)

func ValidatePathParameters(request events.APIGatewayProxyRequest) (string, error) {
	switch len(request.PathParameters) {
	case 0:
		return "", nil
	case 1:
		if pid, found := request.PathParameters["pid"]; found {
			validFormat := regexp.MustCompile(validPidRegex).MatchString(pid)
			if !validFormat {
				return "", errors.New("pid is not formatted correctly")
			}
			return pid, nil
		}
		return "", errors.New("invalid path parameters")
	default:
		return "", errors.New("too many path parameters provided")
	}
}

func ValidateCreatePlayerRequest(requestBody string) (*CreatePlayerRequest, error) {
	var validRequest CreatePlayerRequest

	err := json.Unmarshal([]byte(requestBody), &validRequest)
	if err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(&validRequest); err != nil {
		return nil, err
	}

	return &validRequest, nil
}
