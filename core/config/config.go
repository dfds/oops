package config

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/kelseyhightower/envconfig"
	selfserviceapi "go.dfds.cloud/oops/core/ssu/selfservice-api"
)

type Config struct {
	LogDebug             bool   `json:"logDebug"`
	LogLevel             string `json:"logLevel"`
	AdditionalConfigPath string `json:"additionalConfigPath"`
	Kubernetes           struct {
		ClusterName     string `json:"clusterName"`
		ClusterCa       string `json:"clusterCa"`
		ClusterEndpoint string `json:"clusterEndpoint"`
	} `json:"kubernetes"`
	Enable struct {
		Messaging bool `json:"messaging" default:"true"`
		Operator  bool `json:"operator" default:"true"`
	} `json:"enable"`
	SelfserviceApi selfserviceapi.Config `json:"selfserviceApi"`
	Job            struct {
		Route53Backup struct {
			AssumeRole string `json:"assumeRole"`
			Accounts   string `json:"accounts"`
		} `json:"route53Backup"`
	} `json:"job"`
	BackupLocations []BackupLocation `json:"backupLocations"`
}

func (c *Config) Route53AwsAccounts() []string {
	buf := strings.ReplaceAll(c.Job.Route53Backup.Accounts, " ", "")
	return strings.Split(buf, ",")
}

type BackupLocation struct {
	Name     string                 `json:"name"`
	Provider string                 `json:"provider"`
	Enabled  bool                   `json:"enabled"`
	Spec     map[string]interface{} `json:"spec"`
}

const APP_CONF_PREFIX = "SSU_OOPS"

func LoadConfig() (Config, error) {
	var conf Config
	err := envconfig.Process(APP_CONF_PREFIX, &conf)

	return conf, err
}

func LoadConfigFromJsonFile(path string) (*Config, error) {
	var conf *Config

	buf, err := os.ReadFile(path)
	if err != nil {
		return conf, err
	}

	err = json.Unmarshal(buf, &conf)
	if err != nil {
		return conf, err
	}

	return conf, nil
}

func LocationSpecToType[T any](location BackupLocation) (*T, error) {
	serialised, err := json.Marshal(location.Spec)
	if err != nil {
		return nil, err
	}

	var payload *T
	err = json.Unmarshal(serialised, &payload)
	if err != nil {
		return nil, err
	}

	return payload, nil
}
