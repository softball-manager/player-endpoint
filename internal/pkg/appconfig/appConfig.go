package appconfig

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	cfg "github.com/softball-manager/common/pkg/appconfig"
	"github.com/softball-manager/common/pkg/dynamo"
	"go.uber.org/zap"
)

type AppConfig struct {
	Env       string
	AWSConfig aws.Config
	Logger    *zap.Logger
	TableName string
}

func NewAppConfig(env string, cfg aws.Config, logger *zap.Logger) *AppConfig {
	return &AppConfig{
		Env:       env,
		AWSConfig: cfg,
		Logger:    logger,
	}
}

func (a *AppConfig) GetEnv() string {
	return a.Env
}

func (a *AppConfig) GetAWSConfig() aws.Config {
	return a.AWSConfig
}

func (a *AppConfig) GetLogger() *zap.Logger {
	return a.Logger
}

func (a *AppConfig) ReadEnvVars() {
	a.TableName = cfg.GetEnvVarStringOrDefault("PLAYERTABLENAME", fmt.Sprintf("%s-%s", dynamo.PlayerTableNamePrefix, cfg.LocalEnv))
}
