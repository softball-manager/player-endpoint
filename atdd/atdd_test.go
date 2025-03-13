package atdd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
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

type Feature struct {
	env                string
	tableName          string
	basePlayerEndpoint string

	db *dynamodb.Client

	createPlayerRequest  *request.CreatePlayerRequest
	createPlayerResponse *response.SuccessfulCreateResponse
	getPlayerResponse    *player.Player

	statusCode      int
	pid             string
	playerName      string
	playerPositions []string
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
		createPlayerRequest:  &request.CreatePlayerRequest{},
		createPlayerResponse: &response.SuccessfulCreateResponse{},
		getPlayerResponse:    &player.Player{},
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
	ctx.Given(`^there is a player with the name "([^"]*)" who plays$`, f.thereIsAPlayerWithTheNameWhoPlays)
	ctx.Given(`^there is a player with the name "([^"]*)" who has no defined position$`, f.thereIsAPlayerWithTheNameWhoHasNoDefinedPosition)

	ctx.When(`^I submit a request to create the player$`, f.iSubmitARequestToCreateThePlayer)
	ctx.When(`^I submit a request to get the player$`, f.iSubmitARequestToGetThePlayer)

	ctx.Then(`^the new player item exists in the database$`, f.theNewPlayerItemExistsInTheDatabase)
	ctx.Then(`^I receive a successful response$`, f.iReceiveASuccessfulResponse)
	ctx.Then(`^I receive a bad request response$`, f.iReceiveABadRequestResponse)
	ctx.Then(`^I receive a not found response$`, f.iReceiveNotFoundResponse)
	ctx.Then(`^the get response body is validated$`, f.theGetResponseBodyIsValidated)

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
	positions, err := readPositionsTable(table)
	if err != nil {
		return err
	}

	f.createPlayerRequest.Positions = positions
	return nil
}

func (f *Feature) thereIsAPlayerWithTheNameWhoPlays(ctx context.Context, name string, table *godog.Table) error {
	positions, err := readPositionsTable(table)
	if err != nil {
		return nil
	}

	pid := "Player#GetATDDTest"
	p := player.Player{
		PK:        pid,
		SK:        pid,
		Name:      name,
		Positions: positions,
	}

	err = f.putPlayerInDB(ctx, p)
	if err != nil {
		return err
	}

	f.pid = pid
	f.playerName = name
	f.playerPositions = positions
	return nil
}

func (f *Feature) thereIsAPlayerWithTheNameWhoHasNoDefinedPosition(ctx context.Context, name string) error {
	pid := "Player#GetATDDTestNoPositions"
	p := player.Player{
		PK:        pid,
		SK:        pid,
		Name:      name,
		Positions: []string{},
	}

	err := f.putPlayerInDB(ctx, p)
	if err != nil {
		return err
	}

	f.pid = pid
	f.playerName = name
	f.playerPositions = []string{}
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

	return nil
}

func (f *Feature) iSubmitARequestToGetThePlayer(ctx context.Context) error {
	getUrl := fmt.Sprintf("%s%s", f.basePlayerEndpoint, url.QueryEscape(f.pid))
	getRequest, err := http.NewRequest(http.MethodGet, getUrl, bytes.NewBuffer([]byte{}))
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(getRequest)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(f.getPlayerResponse)
	if err != nil {
		return err
	}

	f.statusCode = resp.StatusCode
	return nil
}

func (f *Feature) iReceiveASuccessfulResponse(ctx context.Context) error {
	if f.statusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code received | excpected: %v, received: %v", http.StatusOK, f.statusCode)
	}

	if !reflect.DeepEqual(*f.createPlayerRequest, request.CreatePlayerRequest{}) {
		f.pid = f.createPlayerResponse.ID
		f.playerName = f.createPlayerRequest.Name
		f.playerPositions = f.createPlayerRequest.Positions
	}

	return nil
}

func (f *Feature) iReceiveABadRequestResponse(ctx context.Context) error {
	if f.statusCode != http.StatusBadRequest {
		return fmt.Errorf("unexpected status code received | excpected: %v, received: %v", http.StatusBadRequest, f.statusCode)
	}
	return nil
}

func (f *Feature) iReceiveNotFoundResponse(ctx context.Context) error {
	if f.statusCode != http.StatusNotFound {
		return fmt.Errorf("unexpected status code received | excpected: %v, received: %v", http.StatusNotFound, f.statusCode)
	}
	return nil
}

func (f *Feature) theNewPlayerItemExistsInTheDatabase(ctx context.Context) error {
	expectedPlayer := player.Player{
		PK:        f.pid,
		SK:        f.pid,
		Name:      f.playerName,
		Positions: f.playerPositions,
		Stats:     []player.Stats{},
	}

	actualPlayer, err := f.getPlayerFromDB(ctx)
	if err != nil {
		return err
	}

	assert.Equal(godog.T(ctx), expectedPlayer, actualPlayer, "the retrieved item does not equal the expected item")

	return nil
}

func (f *Feature) theGetResponseBodyIsValidated(ctx context.Context) error {
	expectedPlayer := player.Player{
		PK:        f.pid,
		SK:        f.pid,
		Name:      f.playerName,
		Positions: f.playerPositions,
	}

	assert.Equal(godog.T(ctx), expectedPlayer, *f.getPlayerResponse, "the retrieved item does not equal the expected item")
	return nil
}

func (f *Feature) getPlayerFromDB(ctx context.Context) (player.Player, error) {
	result, err := f.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(f.tableName),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: f.pid},
			"sk": &types.AttributeValueMemberS{Value: f.pid},
		},
	})
	if err != nil {
		return player.Player{}, err
	}

	var actualPlayer player.Player
	err = attributevalue.UnmarshalMap(result.Item, &actualPlayer)
	if err != nil {
		return player.Player{}, err
	}

	return actualPlayer, nil
}

func (f *Feature) putPlayerInDB(ctx context.Context, p player.Player) error {
	av, err := attributevalue.MarshalMap(p)
	if err != nil {
		return err
	}

	_, err = f.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(f.tableName),
		Item:      av,
	})
	if err != nil {
		return err
	}

	return nil
}

func readPositionsTable(table *godog.Table) ([]string, error) {
	if table.Rows[0].Cells[0].Value != "Positions" {
		return []string{}, errors.New("invalid header value | expected \"Positions\"")
	}

	positions := make([]string, len(table.Rows)-1)
	for i, row := range table.Rows[1:] {
		positions[i] = row.Cells[0].Value
	}

	return positions, nil
}
