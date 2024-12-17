package main

import (
	"context"
	"fmt"
	"net/http"
	"softball-manager/player-endpoint/internal/appconfig"
	"softball-manager/player-endpoint/internal/repository"
	"softball-manager/player-endpoint/internal/request"
	"softball-manager/player-endpoint/internal/response"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/google/uuid"
	commonCfg "github.com/softball-manager/common/pkg/appconfig"
	"github.com/softball-manager/common/pkg/awsconfig"
	"github.com/softball-manager/common/pkg/dynamo"
	"github.com/softball-manager/common/pkg/log"
	"go.uber.org/zap"
)

var playerEndpoint = "player-endpoint"
var dynamoClient *dynamodb.Client
var appCfg *appconfig.AppConfig
var repo *repository.Repository

func init() {
	env := commonCfg.GetEnvironment()

	logger := log.GetLoggerWithEnv(log.InfoLevel, env)
	logger.Sugar().Infof("initializing %s", playerEndpoint)

	cfg, err := awsconfig.GetAWSConfig(context.TODO(), env)
	if err != nil {
		logger.Sugar().Fatalf("Unable to load SDK config: %v", err)
	}

	appCfg = appconfig.NewAppConfig(env, cfg, logger)
	appCfg.ReadEnvVars()

	dynamoClient = dynamo.CreateClient(appCfg)
	repo = repository.NewRespository(context.TODO(), appCfg, dynamoClient)
}

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger := appCfg.SetLogger(log.GetLoggerWithEnv(log.InfoLevel, appCfg.Env))
	logger.Info("recieved event")

	pid, err := request.ValidatePathParameters(req)
	if err != nil {
		logger.Error("error validating path parameters", zap.Error(err))
		return response.CreateBadRequestResponse(), nil
	}

	switch req.HTTPMethod {
	case http.MethodPost:
		if pid == "" {
			return handleCreatePlayer(ctx, req.Body)
		}
		return handleUpdatePlayer(ctx, pid, req.Body)
	case http.MethodGet:
		return handleGetPlayer(ctx, pid)
	default:
		return response.CreateBadRequestResponse(), nil
	}

}

func handleCreatePlayer(ctx context.Context, requestBody string) (events.APIGatewayProxyResponse, error) {
	pid := fmt.Sprintf("%s%s", dynamo.PlayerIDPrefix, uuid.New())
	appCfg.Logger = appCfg.Logger.With(zap.String(log.PlayerIDLogKey, pid))
	logger := appCfg.GetLogger()

	validatedRequest, err := request.ValidateCreatePlayerRequest(requestBody)
	if err != nil {
		logger.Error("error validating request", zap.Error(err))
		return response.CreateBadRequestResponse(), nil
	}

	err = repo.PutPlayer(pid, validatedRequest.Name, validatedRequest.Positions)
	if err != nil {
		logger.Error("error putting player into db", zap.Error(err))
		return response.CreateInternalServerErrorResponse(), nil
	}

	return response.CreateSuccessfulCreatePlayerResponse(pid), nil
}

func handleUpdatePlayer(ctx context.Context, pid string, requestBody string) (events.APIGatewayProxyResponse, error) {
	return response.CreateSuccesfulUpdatePlayerResponse(), nil
}

func handleGetPlayer(ctx context.Context, pid string) (events.APIGatewayProxyResponse, error) {
	appCfg.Logger = appCfg.Logger.With(zap.String(log.PlayerIDLogKey, pid))
	logger := appCfg.GetLogger()
	logger.Info("request validated")

	p, err := repo.GetPlayer(pid)
	if err != nil {
		logger.Error("error getting player from db", zap.Error(err))
		return response.CreateInternalServerErrorResponse(), nil
	}

	if p.PK == "" {
		return response.CreateResourceNotFoundResponse(), nil
	}

	return response.CreateSuccessfulGetPlayerResponse(p), nil
}

func main() {
	lambda.Start(handler)
}
