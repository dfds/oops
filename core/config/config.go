package config

import (
	"github.com/kelseyhightower/envconfig"
	selfserviceapi "go.dfds.cloud/oops/core/ssu/selfservice-api"
)

type Config struct {
	LogDebug   bool   `json:"logDebug"`
	LogLevel   string `json:"logLevel"`
	Kubernetes struct {
		ClusterName     string `json:"clusterName"`
		ClusterCa       string `json:"clusterCa"`
		ClusterEndpoint string `json:"clusterEndpoint"`
	} `json:"kubernetes"`
	Enable struct {
		Messaging bool `json:"messaging" default:"true"`
		Operator  bool `json:"operator" default:"true"`
	} `json:"enable"`
	SelfserviceApi selfserviceapi.Config `json:"selfserviceApi"`
}

const APP_CONF_PREFIX = "SSU_OOPS"

func LoadConfig() (Config, error) {
	var conf Config
	err := envconfig.Process(APP_CONF_PREFIX, &conf)

	return conf, err
}
