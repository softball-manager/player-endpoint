package repository

import (
	"context"
	"softball-manager/player-endpoint/internal/pkg/appconfig"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/softball-manager/common/pkg/log"
	"github.com/softball-manager/common/pkg/player"
	"go.uber.org/zap"
)

type Repository struct {
	Ctx       context.Context
	AppConfig *appconfig.AppConfig
	Client    *dynamodb.Client
	TableName string
	logger    *zap.Logger
}

func NewRespository(ctx context.Context, cfg *appconfig.AppConfig, client *dynamodb.Client) *Repository {
	return &Repository{
		Ctx:       ctx,
		AppConfig: cfg,
		Client:    client,
		TableName: cfg.TableName,
		logger:    cfg.GetLogger().With(zap.String(log.TableNameLogKey, cfg.TableName)),
	}
}

func (r *Repository) PutPlayer(pid string, name string, positions []string) error {
	p := player.Player{
		PK:        pid,
		SK:        pid,
		Name:      name,
		Positions: positions,
		Stats:     []player.Stats{},
	}

	r.logger.Info("marshalling player struct")
	av, err := attributevalue.MarshalMap(p)
	if err != nil {
		return err
	}

	r.logger.Info("inserting item into db", zap.Any("item", av))
	_, err = r.Client.PutItem(r.Ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.TableName),
		Item:      av,
	})
	if err != nil {
		return err
	}
	r.logger.Info("successfully inserted item")

	return nil
}

func (r *Repository) GetPlayer(pid string) (player.Player, error) {
	r.logger.Info("getting item from db", zap.Any(log.PlayerIDLogKey, pid))
	result, err := r.Client.GetItem(r.Ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.TableName),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: pid},
			"sk": &types.AttributeValueMemberS{Value: pid},
		},
	})
	if err != nil {
		return player.Player{}, err
	}
	r.logger.Info("successfully retrieved item")

	var p player.Player
	err = attributevalue.UnmarshalMap(result.Item, &p)
	if err != nil {
		return player.Player{}, err
	}

	return p, err
}
