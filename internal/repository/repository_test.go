package repository

import (
	"context"
	"errors"
	"softball-manager/player-endpoint/internal/appconfig"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/softball-manager/common/pkg/player"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type MockDB struct{}

func (m *MockDB) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	if av, found := params.Item["pk"]; found {
		if pk, ok := av.(*types.AttributeValueMemberS); ok {
			switch pk.Value {
			case "Player#123":
				return nil, nil
			case "Error":
				return nil, errors.New("An error occured during Put Item")
			}
		}
	}
	return nil, errors.New("default mock put item error")
}

func (m *MockDB) GetItem(ctx context.Context, params *dynamodb.GetItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error) {
	if av, found := params.Key["pk"]; found {
		if pk, ok := av.(*types.AttributeValueMemberS); ok {
			switch pk.Value {
			case "Player#123":
				return &dynamodb.GetItemOutput{
					Item: map[string]types.AttributeValue{
						"pk":        &types.AttributeValueMemberS{Value: pk.Value},
						"sk":        &types.AttributeValueMemberS{Value: pk.Value},
						"name":      &types.AttributeValueMemberS{Value: "testName"},
						"positions": &types.AttributeValueMemberSS{Value: []string{"1B", "SS", "CF"}},
					},
				}, nil
			case "Get Item Error":
				return nil, errors.New("An error occured during Get Item")
			case "Unmarshal Error":
				return &dynamodb.GetItemOutput{
					Item: map[string]types.AttributeValue{
						"pk":   &types.AttributeValueMemberS{Value: pk.Value},
						"sk":   &types.AttributeValueMemberS{Value: pk.Value},
						"name": &types.AttributeValueMemberBOOL{Value: true},
					},
				}, nil
			}
		}
	}
	return nil, nil
}

var (
	ac   = appconfig.NewAppConfig("TEST", *aws.NewConfig(), zap.Must(zap.NewDevelopmentConfig().Build()))
	repo = NewRespository(context.Background(), ac, &MockDB{})
)

func TestPutPlayer(t *testing.T) {
	tests := map[string]struct {
		pid           string
		name          string
		positions     []string
		expectedError string
	}{
		"Happy path - Player added to db": {
			pid:       "Player#123",
			name:      "testName",
			positions: []string{"1B", "2B", "3B"},
		},
		"Sad path - DB PutItem operation error": {
			pid:           "Error",
			expectedError: "An error occured during Put Item",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := repo.PutPlayer(tc.pid, tc.name, tc.positions)

			if tc.expectedError != "" {
				assert.Equal(t, tc.expectedError, err.Error(), "Recieved wrong error")
			} else {
				assert.Nil(t, err, "Expected error to be nil")
			}
		})
	}
}

func TestGetPlayer(t *testing.T) {
	tests := map[string]struct {
		pid            string
		expectedPlayer player.Player
		expectedError  string
	}{
		"Happy path - Player added to db": {
			pid: "Player#123",
			expectedPlayer: player.Player{
				PK:        "Player#123",
				SK:        "Player#123",
				Name:      "testName",
				Positions: []string{"1B", "SS", "CF"},
			},
		},
		"Sad path - DB PutItem operation error": {
			pid:           "Get Item Error",
			expectedError: "An error occured during Get Item",
		},
		"Sad path - Unmarshal error": {
			pid:           "Unmarshal Error",
			expectedError: "unmarshal failed, cannot unmarshal bool into Go value type string",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			player, err := repo.GetPlayer(tc.pid)

			if tc.expectedError != "" {
				assert.Equal(t, tc.expectedError, err.Error(), "Recieved wrong error")
				assert.Empty(t, player, "Expected empty player object")
			} else {
				assert.Nil(t, err, "Expected error to be nil")
				assert.Equal(t, tc.expectedPlayer, player)
			}
		})
	}
}
