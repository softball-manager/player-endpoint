package request

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/go-playground/validator/v10"
	"github.com/softball-manager/common/pkg/dynamo"
)

type CreatePlayerRequest struct {
	Name      string   `json:"name" validate:"required"`
	Positions []string `json:"positions"`
}

type UpdatePlayerRequest struct {
	Name      string   `json:"name"`
	Positions []string `json:"positions"`
}

var (
	playerIDPathParameterPrefix = "Player%23"
	validPidRegex               = fmt.Sprintf(`^%s[a-zA-Z0-9-]+$`, dynamo.PlayerIDPrefix)
)

func ValidatePathParameters(request events.APIGatewayProxyRequest) (string, error) {
	switch len(request.PathParameters) {
	case 0:
		return "", nil
	case 1:
		if pid, found := request.PathParameters["pid"]; found {
			pid = strings.Replace(pid, playerIDPathParameterPrefix, dynamo.PlayerIDPrefix, 1)
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

	decoder := json.NewDecoder(bytes.NewReader([]byte(requestBody)))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&validRequest)
	if err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(&validRequest); err != nil {
		return nil, err
	}

	return &validRequest, nil
}

func ValidateUpdatePlayerRequest(requestBody string) (*UpdatePlayerRequest, error) {
	var validRequest UpdatePlayerRequest

	decoder := json.NewDecoder(bytes.NewReader([]byte(requestBody)))
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&validRequest)
	if err != nil {
		return nil, err
	}

	validate := validator.New()
	if err := validate.Struct(&validRequest); err != nil {
		return nil, err
	}

	return &validRequest, nil
}
