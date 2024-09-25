package main

import (
	"context"
	"fmt"
	"softball-manager/endpoint-template/internal/pkg/appconfig"
	"softball-manager/endpoint-template/internal/pkg/repository"
	"softball-manager/endpoint-template/internal/pkg/request"
	"softball-manager/endpoint-template/internal/pkg/response"

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

var putPlayerEndpoint = "put-player"
var dynamoClient *dynamodb.Client
var appCfg *appconfig.AppConfig

func init() {
	env := commonCfg.GetEnvironment()

	logger := log.GetLogger("info").With(zap.String(log.EnvLogKey, env))
	logger.Sugar().Infof("initializing %s endpoint", putPlayerEndpoint)

	cfg, err := awsconfig.GetAWSConfig(context.TODO(), env)
	if err != nil {
		logger.Sugar().Fatalf("Unable to load SDK config: %v", err)
	}

	appCfg = appconfig.NewAppConfig(env, cfg, logger)
	appCfg.ReadEnvVars()

	dynamoClient = dynamo.CreateClient(appCfg)

}

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	pid := fmt.Sprintf("%s%s", dynamo.PlayerIDPrefix, uuid.New())
	appCfg.Logger = appCfg.Logger.With(zap.String(log.PlayerIDLogKey, pid))
	logger := appCfg.GetLogger()
	logger.Info("recieved event")

	validatedRequest, err := request.ValidateRequest(req)
	if err != nil {
		logger.Error("error validating request", zap.Error(err))
		return response.CreateBadRequestResponse(), nil
	}

	repository := repository.NewRespository(ctx, appCfg, dynamoClient, fmt.Sprintf("%s-%s", dynamo.PlayerTableNamePrefix, appCfg.GetEnv()))
	err = repository.PutPlayer(pid, validatedRequest.Name, validatedRequest.Positions)
	if err != nil {
		logger.Error("error putting player into db", zap.Error(err))
		return response.CreateInternalServerErrorResponse(), nil
	}

	return response.CreateSuccessfulResponse(pid), nil
}

func main() {
	lambda.Start(handler)
}
