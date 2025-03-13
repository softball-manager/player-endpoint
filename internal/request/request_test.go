package request

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func TestValidatePathParameters(t *testing.T) {
	tests := map[string]struct {
		pathParameters       map[string]string
		expectedPid          string
		expectedErrorMessage string
	}{
		"Happy path - No path parameters provided": {
			pathParameters: map[string]string{},
			expectedPid:    "",
		},
		"Happy path - Player# prefix": {
			pathParameters: map[string]string{
				"pid": "Player#123",
			},
			expectedPid: "Player#123",
		},
		"Happy path - Player%23 prefix": {
			pathParameters: map[string]string{
				"pid": "Player%23123",
			},
			expectedPid: "Player#123",
		},
		"Sad path - path parameter not formatted correctly": {
			pathParameters: map[string]string{
				"pid": "Player!123",
			},
			expectedErrorMessage: "pid is not formatted correctly",
		},
		"Sad path - too many path parameters": {
			pathParameters: map[string]string{
				"pid":          "Player#123",
				"antoherParam": "test",
			},
			expectedErrorMessage: "too many path parameters provided",
		},
		"Sad path - pid not in path parameters": {
			pathParameters: map[string]string{
				"tid": "Player#123",
			},
			expectedErrorMessage: "invalid path parameters",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			apiGatewayRequest := events.APIGatewayProxyRequest{
				PathParameters: tc.pathParameters,
			}

			pid, err := ValidatePathParameters(apiGatewayRequest)

			if tc.expectedErrorMessage != "" {
				assert.Equal(t, tc.expectedErrorMessage, err.Error(), "incorrect error received")
				assert.Empty(t, pid, "expected an empty pid")
			} else {
				assert.Nil(t, err, "expected no error")
				assert.Equal(t, tc.expectedPid, pid, "actual pid value does not match expected")
			}
		})
	}
}

func TestValidateUpdatePlayerRequest(t *testing.T) {
	tests := map[string]struct {
		requestBody     string
		expectedRequest CreatePlayerRequest
		expectError     bool
	}{
		"Happy path - Name and Positions provided": {
			requestBody: "{\"name\": \"unit test 1\", \"positions\": [\"SC\", \"CF\", \"2B\"]}",
			expectedRequest: CreatePlayerRequest{
				Name:      "unit test 1",
				Positions: []string{"SC", "CF", "2B"},
			},
		},
		"Happy path - Only Name provided": {
			requestBody: "{\"name\": \"unit test 2\"}",
			expectedRequest: CreatePlayerRequest{
				Name: "unit test 2",
			},
		},
		"Sad path - Only Positions provided": {
			requestBody: "{\"positions\": [\"SC\", \"CF\", \"2B\"]}",
			expectError: true,
		},
		"Sad path - Too many fields provided": {
			requestBody: "{\"name\": \"unit test 4\", \"positions\": [\"SC\", \"CF\", \"2B\"], \"extraField\": \"testing\"}",
			expectError: true,
		},
		"Sad path - wrong type provided for name": {
			requestBody: "{\"name\": 654}",
			expectError: true,
		},
		"Sad path - wrong type provided for positions": {
			requestBody: "{\"name\": \"unit test 6\", \"positions\": \"not an array\"}",
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			createPlayerRequest, err := ValidateCreatePlayerRequest(tc.requestBody)

			if tc.expectError {
				assert.NotNil(t, err, "expected an error")
				assert.Nil(t, createPlayerRequest, "expected the returned request to be nil")
			} else {
				assert.Nil(t, err, "expected no error")
				assert.Equal(t, tc.expectedRequest, *createPlayerRequest)
			}
		})
	}
}

func TestValidateCreatePlayerRequest(t *testing.T) {
	tests := map[string]struct {
		requestBody     string
		expectedRequest UpdatePlayerRequest
		expectError     bool
	}{
		"Happy path - Name and Positions provided": {
			requestBody: "{\"name\": \"unit test 1\", \"positions\": [\"SC\", \"CF\", \"2B\"]}",
			expectedRequest: UpdatePlayerRequest{
				Name:      "unit test 1",
				Positions: []string{"SC", "CF", "2B"},
			},
		},
		"Happy path - Only Name provided": {
			requestBody: "{\"name\": \"unit test 2\"}",
			expectedRequest: UpdatePlayerRequest{
				Name: "unit test 2",
			},
		},
		"Happy path - Only Positions provided": {
			requestBody: "{\"positions\": [\"SC\", \"CF\", \"2B\"]}",
			expectedRequest: UpdatePlayerRequest{
				Positions: []string{"SC", "CF", "2B"},
			},
		},
		"Sad path - Too many fields provided": {
			requestBody: "{\"name\": \"unit test 4\", \"positions\": [\"SC\", \"CF\", \"2B\"], \"extraField\": \"testing\"}",
			expectError: true,
		},
		"Sad path - wrong type provided for name": {
			requestBody: "{\"name\": 654}",
			expectError: true,
		},
		"Sad path - wrong type provided for positions": {
			requestBody: "{\"name\": \"unit test 6\", \"positions\": \"not an array\"}",
			expectError: true,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			updatePlayerRequest, err := ValidateUpdatePlayerRequest(tc.requestBody)

			if tc.expectError {
				assert.NotNil(t, err, "expected an error")
				assert.Nil(t, updatePlayerRequest, "expected the returned request to be nil")
			} else {
				assert.Nil(t, err, "expected no error")
				assert.Equal(t, tc.expectedRequest, *updatePlayerRequest)
			}
		})
	}
}
