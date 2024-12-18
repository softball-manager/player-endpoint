package atdd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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
	"github.com/softball-manager/common/pkg/player"
	"github.com/stretchr/testify/assert"
)

type (
	dynamoClientKey struct{}
	envKey          struct{}
	tableNameKey    struct{}
	baseUrlKey      struct{}
	pidKey          struct{}
	requestKey      struct{}
	responseKey     struct{}
	statusCodeKey   struct{}
)

var (
	pidReplacement = "{{pid}}"

	playerEndpoint = "player/"

	requestFilePath          = "./resources/requests/"
	expectedResponseFilePath = "./resources/expectedResponses/"
)

func beforeScenario(ctx context.Context, sc *godog.Scenario) (context.Context, error) {
	env := cfg.GetEnvironment()
	awsConfig, err := awsconfig.GetAWSConfig(ctx, env)
	if err != nil {
		return ctx, err
	}

	var db *dynamodb.Client
	if env == cfg.LocalEnv {
		db = dynamodb.NewFromConfig(awsConfig, func(o *dynamodb.Options) {
			o.BaseEndpoint = aws.String("http://localhost:8000")
		})
	} else {
		db = dynamodb.NewFromConfig(awsConfig)
	}

	ctx = context.WithValue(ctx, envKey{}, env)
	ctx = context.WithValue(ctx, dynamoClientKey{}, db)
	ctx = context.WithValue(ctx, tableNameKey{}, fmt.Sprintf("player-table-%s", env))

	return ctx, nil
}

func afterScenario(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
	pid, ok := ctx.Value(pidKey{}).(string)
	if !ok {
		return ctx, errors.New("could not retrieve pid from context")
	}

	db, ok := ctx.Value(dynamoClientKey{}).(*dynamodb.Client)
	if !ok {
		return ctx, errors.New("could not retrieve dyanmo client from context")
	}

	tableName, ok := ctx.Value(tableNameKey{}).(string)
	if !ok {
		return ctx, errors.New("could not retrieve table name from context")
	}

	db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: pid},
			"sk": &types.AttributeValueMemberS{Value: pid},
		},
	})

	return ctx, nil
}

func createRequest(ctx context.Context, filename string) (context.Context, error) {
	reqBody, err := os.ReadFile(fmt.Sprintf("%s%s", requestFilePath, filename))
	if err != nil {
		return ctx, err
	}

	env, ok := ctx.Value(envKey{}).(string)
	if !ok {
		return ctx, errors.New("could not retrieve env from context")
	}

	switch env {
	case cfg.LocalEnv:
		ctx = context.WithValue(ctx, baseUrlKey{}, "http://localhost:3000/")
	default:
		ctx = context.WithValue(ctx, baseUrlKey{}, "http://todo/")
	}

	ctx = context.WithValue(ctx, requestKey{}, reqBody)
	return ctx, nil
}

func makeCreateRequest(ctx context.Context) (context.Context, error) {
	baseUrl, ok := ctx.Value(baseUrlKey{}).(string)
	if !ok {
		return ctx, errors.New("could not retrieve base url from context")
	}
	url := fmt.Sprintf("%s%s", baseUrl, playerEndpoint)

	body, ok := ctx.Value(requestKey{}).([]byte)
	if !ok {
		return ctx, errors.New("could not retrieve request body from context")
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return ctx, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return ctx, err
	}
	defer resp.Body.Close()

	successfulResp := &response.SuccessfulCreatePlayerResponse{}
	err = json.NewDecoder(resp.Body).Decode(successfulResp)
	if err != nil {
		return ctx, err
	}

	ctx = context.WithValue(ctx, responseKey{}, successfulResp)
	ctx = context.WithValue(ctx, statusCodeKey{}, resp.StatusCode)
	ctx = context.WithValue(ctx, pidKey{}, successfulResp.PID)

	return ctx, nil
}

func getResponse(ctx context.Context, filename string) error {
	resp, ok := ctx.Value(responseKey{}).(*response.SuccessfulCreatePlayerResponse)
	if !ok {
		return errors.New("could not retrieve response from context")
	}

	statusCode, ok := ctx.Value(statusCodeKey{}).(int)
	if !ok {
		return errors.New("could not retrieve status code from context")
	}

	pid, ok := ctx.Value(pidKey{}).(string)
	if !ok {
		return errors.New("could not retrieve pid from context")
	}

	expectedResponseBytes, err := os.ReadFile(fmt.Sprintf("%s%s", expectedResponseFilePath, filename))
	if err != nil {
		return err
	}

	expectedResponse := &response.SuccessfulCreatePlayerResponse{}
	err = json.Unmarshal(expectedResponseBytes, expectedResponse)
	if err != nil {
		return err
	}
	expectedResponse.PID = strings.ReplaceAll(expectedResponse.PID, pidReplacement, pid)

	assert.Equal(godog.T(ctx), expectedResponse, resp)
	assert.Equal(godog.T(ctx), http.StatusOK, statusCode, "Expected status: %d | Actual status: %d", http.StatusOK, statusCode)

	return nil
}

func validateNewPlayerInDB(ctx context.Context) error {
	reqBytes, ok := ctx.Value(requestKey{}).([]byte)
	if !ok {
		return errors.New("could not retrieve request from context")
	}

	pid, ok := ctx.Value(pidKey{}).(string)
	if !ok {
		return errors.New("could not retrieve pid from context")
	}

	db, ok := ctx.Value(dynamoClientKey{}).(*dynamodb.Client)
	if !ok {
		return errors.New("could not retrieve dyanmo client from context")
	}

	tableName, ok := ctx.Value(tableNameKey{}).(string)
	if !ok {
		return errors.New("could not retrieve table name from context")
	}

	reqBody := &request.CreatePlayerRequest{}
	err := json.Unmarshal(reqBytes, reqBody)
	if err != nil {
		return err
	}

	expectedPlayer := player.Player{
		PK:        pid,
		SK:        pid,
		Name:      reqBody.Name,
		Positions: reqBody.Positions,
		Stats:     []player.Stats{},
	}

	result, err := db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: pid},
			"sk": &types.AttributeValueMemberS{Value: pid},
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

func InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.Before(beforeScenario)

	ctx.Given(`^I have a post request with body ([a-zA-Z0-9._-]+\.json)$`, createRequest)
	ctx.When(`^I call the post endpoint to create a player$`, makeCreateRequest)
	ctx.Then(`^the response should match ([a-zA-Z0-9._-]+\.json)$`, getResponse)
	ctx.Then(`the new player item exists in the database`, validateNewPlayerInDB)

	ctx.After(afterScenario)
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
