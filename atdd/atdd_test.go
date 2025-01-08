package atdd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"softball-manager/player-endpoint/internal/request"
	"softball-manager/player-endpoint/internal/response"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/cucumber/godog"
	cfg "github.com/softball-manager/common/pkg/appconfig"
	"github.com/softball-manager/common/pkg/awsconfig"
	"github.com/softball-manager/common/pkg/dynamo"
	"github.com/softball-manager/common/pkg/player"
	"github.com/stretchr/testify/assert"
)

const (
	localhost      = "http://localhost"
	playerEndpoint = "player/"
	devUrl         = "TODO"
)

var (
	pidReplacement = "{{pid}}"

	requestFilePath          = "./resources/requests/"
	expectedResponseFilePath = "./resources/expectedResponses/"
)

type Feature struct {
	env                string
	tableName          string
	basePlayerEndpoint string
	pid                string

	db *dynamodb.Client

	createPlayerRequest     *request.CreatePlayerRequest
	createPlayerResponse    *response.SuccessfulCreatePlayerResponse
	createPlayerHttpRequest *http.Request
	statusCode              int
}

func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t, // Testing instance that will run subtests.
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	f := Feature{
		createPlayerRequest:     &request.CreatePlayerRequest{},
		createPlayerResponse:    &response.SuccessfulCreatePlayerResponse{},
		createPlayerHttpRequest: &http.Request{},
	}

	ctx.Before(func(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
		f.env = cfg.GetEnvironment()
		awsConfig, err := awsconfig.GetAWSConfig(ctx, f.env)
		if err != nil {
			return ctx, err
		}

		if f.env == cfg.LocalEnv {
			f.db = dynamodb.NewFromConfig(awsConfig, func(o *dynamodb.Options) {
				o.BaseEndpoint = aws.String(fmt.Sprintf("%s:8000", localhost))
			})
		} else {
			f.db = dynamodb.NewFromConfig(awsConfig)
		}

		switch f.env {
		case cfg.LocalEnv:
			f.basePlayerEndpoint = fmt.Sprintf("%s:3000/%s", localhost, playerEndpoint)
		default:
			f.basePlayerEndpoint = fmt.Sprintf("%s/%s", devUrl, playerEndpoint)
		}

		f.tableName = fmt.Sprintf("%s-%s", dynamo.PlayerTableNamePrefix, f.env)

		return ctx, nil
	})

	ctx.Given(`^I have a post request with body ([a-zA-Z0-9._-]+\.json)$`, f.createRequest)
	ctx.When(`^I call the post endpoint to create a player$`, f.makeCreateRequest)
	ctx.Then(`^the response should match ([a-zA-Z0-9._-]+\.json)$`, f.getResponse)
	ctx.Then(`the new player item exists in the database`, f.validateNewPlayerInDB)

	ctx.After(func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
		if f.pid == "" {
			return ctx, nil
		}

		f.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
			TableName: aws.String(f.tableName),
			Key: map[string]types.AttributeValue{
				"pk": &types.AttributeValueMemberS{Value: f.pid},
				"sk": &types.AttributeValueMemberS{Value: f.pid},
			},
		})

		return ctx, nil
	})
}

func (f *Feature) createRequest(ctx context.Context, filename string) error {
	filePath := fmt.Sprintf("%s%s", requestFilePath, filename)
	reqBody, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	err = json.Unmarshal(reqBody, f.createPlayerRequest)
	if err != nil {
		return err
	}

	f.createPlayerHttpRequest, err = http.NewRequest(http.MethodPost, f.basePlayerEndpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}
	return nil
}

func (f *Feature) makeCreateRequest(ctx context.Context) error {
	client := &http.Client{}
	resp, err := client.Do(f.createPlayerHttpRequest)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(f.createPlayerResponse)
	if err != nil {
		return err
	}

	f.statusCode = resp.StatusCode
	f.pid = f.createPlayerResponse.PID

	return nil
}

func (f *Feature) getResponse(ctx context.Context, filename string) error {
	expectedResponseBytes, err := os.ReadFile(fmt.Sprintf("%s%s", expectedResponseFilePath, filename))
	if err != nil {
		return err
	}

	expectedResponse := &response.SuccessfulCreatePlayerResponse{}
	err = json.Unmarshal(expectedResponseBytes, expectedResponse)
	if err != nil {
		return err
	}
	expectedResponse.PID = strings.ReplaceAll(expectedResponse.PID, pidReplacement, f.pid)

	assert.Equal(godog.T(ctx), expectedResponse, f.createPlayerResponse)
	assert.Equal(godog.T(ctx), http.StatusOK, f.statusCode, "Expected status: %d | Actual status: %d", http.StatusOK, f.statusCode)

	return nil
}

func (f *Feature) validateNewPlayerInDB(ctx context.Context) error {
	expectedPlayer := player.Player{
		PK:        f.pid,
		SK:        f.pid,
		Name:      f.createPlayerRequest.Name,
		Positions: f.createPlayerRequest.Positions,
		Stats:     []player.Stats{},
	}

	result, err := f.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(f.tableName),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: f.pid},
			"sk": &types.AttributeValueMemberS{Value: f.pid},
		},
	})
	if err != nil {
		return err
	}

	var actualPlayer player.Player
	err = attributevalue.UnmarshalMap(result.Item, &actualPlayer)
	if err != nil {
		return err
	}

	assert.Equal(godog.T(ctx), expectedPlayer, actualPlayer, "the retrieve item does not equal the expected item")

	return nil
}
