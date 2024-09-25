package repository

import (
	"context"
	"softball-manager/endpoint-template/internal/pkg/appconfig"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/softball-manager/common/pkg/log"
	"github.com/softball-manager/common/pkg/player"
	"go.uber.org/zap"
)

type Repository struct {
	Ctx       context.Context
	AppConfig *appconfig.AppConfig
	Client    *dynamodb.Client
	TableName string
}

func NewRespository(ctx context.Context, cfg *appconfig.AppConfig, client *dynamodb.Client, tableName string) *Repository {
	return &Repository{
		Ctx:       ctx,
		AppConfig: cfg,
		Client:    client,
		TableName: tableName,
	}
}

func (r *Repository) PutPlayer(pk string, name string, positions []string) error {
	logger := r.AppConfig.GetLogger().With(zap.String(log.TableNameLogKey, r.TableName))
	p := player.Player{
		PK:        pk,
		Name:      name,
		Positions: positions,
		Stats:     []player.Stats{},
	}

	logger.Info("marshalling player struct")
	av, err := attributevalue.MarshalMap(p)
	if err != nil {
		return err
	}

	logger.Info("inserting item into db", zap.Any("item", av))
	_, err = r.Client.PutItem(r.Ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.TableName),
		Item:      av,
	})
	if err != nil {
		return err
	}
	logger.Info("successfully inserted item")

	return nil
}
