package atdd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"softball-manager/player-endpoint/internal/request"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/cucumber/godog"
	cfg "github.com/softball-manager/common/pkg/appconfig"
	"github.com/softball-manager/common/pkg/awsconfig"
	"github.com/softball-manager/common/pkg/dynamo"
	"github.com/softball-manager/common/pkg/player"
	"github.com/softball-manager/common/pkg/response"
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
	createPlayerResponse    *response.SuccessfulCreateResponse
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

	if testing.Short() {
		t.Skip()
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	f := Feature{
		createPlayerRequest:     &request.CreatePlayerRequest{},
		createPlayerResponse:    &response.SuccessfulCreateResponse{},
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

	ctx.Given(`^I want to create a player with the name "([^"]*)"$`, f.iWantToCreateAPlayerWithName)
	ctx.Given(`^the player plays the following positions$`, f.thePlayerPlaysTheFollowingPositions)
	ctx.When(`^I submit a request to create the player$`, f.iSubmitARequestToCreateThePlayer)
	ctx.Then(`^the new player item exists in the database$`, f.theNewPlayerItemExistsInTheDatabase)

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

func (f *Feature) iWantToCreateAPlayerWithName(name string) error {
	f.createPlayerRequest.Name = name
	return nil
}

func (f *Feature) thePlayerPlaysTheFollowingPositions(table *godog.Table) error {
	if table.Rows[0].Cells[0].Value != "Positions" {
		return errors.New("invalid header value | expected \"Positions\"")
	}

	f.createPlayerRequest.Positions = make([]string, len(table.Rows)-1)
	for i, row := range table.Rows[1:] {
		f.createPlayerRequest.Positions[i] = row.Cells[0].Value
	}

	return nil
}

func (f *Feature) iSubmitARequestToCreateThePlayer(ctx context.Context) error {
	client := &http.Client{}

	reqBody, err := json.Marshal(f.createPlayerRequest)
	if err != nil {
		return err
	}

	createRequest, err := http.NewRequest(http.MethodPost, f.basePlayerEndpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}

	resp, err := client.Do(createRequest)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	dec.DisallowUnknownFields()
	err = dec.Decode(f.createPlayerResponse)
	if err != nil {
		return err
	}

	f.statusCode = resp.StatusCode
	f.pid = f.createPlayerResponse.ID

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code received | excpected: %v, received: %v", http.StatusOK, resp.StatusCode)
	}

	return nil
}

func (f *Feature) theNewPlayerItemExistsInTheDatabase(ctx context.Context) error {
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
