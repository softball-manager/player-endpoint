package appconfig

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

var (
	env       = "TEST"
	awsConfig = aws.NewConfig()
	logger    = zap.Must(zap.NewDevelopmentConfig().Build())
	ac        = NewAppConfig(env, *awsConfig, logger)
)

func TestGetEnv(t *testing.T) {
	assert.Equal(t, env, ac.GetEnv())
}

func TestGetAWSConfig(t *testing.T) {
	assert.Equal(t, *awsConfig, ac.GetAWSConfig())
}

func TestGetLogger(t *testing.T) {
	assert.Equal(t, logger, ac.GetLogger())
}

func TestSetLogger(t *testing.T) {
	newLogger := logger.With(zap.String("testKey", "testVal"))
	ac.SetLogger(newLogger)

	assert.Equal(t, newLogger, ac.Logger)
}

func TestReadEnvVars(t *testing.T) {
	os.Setenv("PLAYER_TABLE_NAME", "myTestTable")
	ac.ReadEnvVars()

	assert.Equal(t, "myTestTable", ac.TableName)
}
