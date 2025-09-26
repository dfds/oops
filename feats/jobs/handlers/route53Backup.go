package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsHttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/route53"
	route53Types "github.com/aws/aws-sdk-go-v2/service/route53/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
	oopsAws "go.dfds.cloud/oops/core/aws"
	"go.dfds.cloud/oops/core/config"
	"go.dfds.cloud/oops/core/logging"
	"go.dfds.cloud/oops/core/util"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

func Route53Backup(ctx context.Context) error {
	logging.Logger.Info("Taking backup of Route53 zones")

	conf, err := config.LoadConfig()
	if err != nil {
		return err
	}

	accs := conf.Route53AwsAccounts()

	sessions, err := AssumeRoleForAccounts(ctx, accs, conf.Job.Route53Backup.AssumeRole)
	if err != nil {
		return err
	}

	recordsByAccountAndZone, err := fetchHostedZones(ctx, sessions)
	if err != nil {
		return err
	}

	serialised, err := json.MarshalIndent(recordsByAccountAndZone, "", "  ")
	if err != nil {
		return err
	}

	err = os.MkdirAll("zones", 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile("zones/records.json", serialised, 0644)
	if err != nil {
		return err
	}

	for acc, zones := range recordsByAccountAndZone {
		for name, zone := range zones {
			zoneFileContent, err := oopsAws.GenerateZoneFile(zone, name)
			if err != nil {
				return err
			}

			dirPath := fmt.Sprintf("zones/%s", acc)

			err = os.MkdirAll(dirPath, 0755)
			if err != nil {
				return err
			}

			err = os.WriteFile(fmt.Sprintf("%s/%s-%s.zone", dirPath, acc, name), []byte(zoneFileContent), 0644)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func AssumeRole(ctx context.Context, roleArn string) (*types.Credentials, error) {
	cfg, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion("eu-west-1"), awsConfig.WithHTTPClient(CreateHttpClientWithoutKeepAlive()))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	stsClient := sts.NewFromConfig(cfg)

	roleSessionName := "oops"

	assumedRole, err := stsClient.AssumeRole(context.TODO(), &sts.AssumeRoleInput{RoleArn: &roleArn, RoleSessionName: &roleSessionName})
	if err != nil {
		log.Printf("unable to assume role %s, %v", roleArn, err)
		return nil, err
	}

	return assumedRole.Credentials, nil
}

func fetchHostedZones(ctx context.Context, sessions map[string]AwsSession) (map[string]map[string][]route53Types.ResourceRecordSet, error) {
	payload := make(map[string]map[string][]route53Types.ResourceRecordSet)
	var maxConcurrentOps int64 = 30
	var waitGroup sync.WaitGroup
	payloadMutex := &sync.Mutex{}
	sem := semaphore.NewWeighted(maxConcurrentOps)

	for _, session := range sessions {
		waitGroup.Add(1)
		sessionWg := session
		go func() {
			sem.Acquire(ctx, 1)
			defer sem.Release(1)
			defer waitGroup.Done()
			logging.Logger.Info(fmt.Sprintf("Fetching hosted zones for account %s\n", sessionWg.AccountId))
			route53Client := route53.NewFromConfig(sessionWg.SessionConfig)

			respZones, err := route53Client.ListHostedZones(ctx, &route53.ListHostedZonesInput{})
			if err != nil {
				logging.Logger.Error("Failed to list hosted zones", zap.Error(err))
				return
			}

			if _, ok := payload[sessionWg.AccountId]; !ok {
				payloadMutex.Lock()
				payload[sessionWg.AccountId] = make(map[string][]route53Types.ResourceRecordSet)
				payloadMutex.Unlock()
			}

			for _, zone := range respZones.HostedZones {
				if _, ok := payload[sessionWg.AccountId][*zone.Name]; !ok {
					payloadMutex.Lock()
					payload[sessionWg.AccountId][*zone.Name] = []route53Types.ResourceRecordSet{}
					payloadMutex.Unlock()
				}

				pag := route53.NewListResourceRecordSetsPaginator(route53Client, &route53.ListResourceRecordSetsInput{HostedZoneId: zone.Id})
				for pag.HasMorePages() {
					recordsResp, err := pag.NextPage(ctx)
					if err != nil {
						logging.Logger.Error("Failed to paginate hosted zone", zap.Error(err))
						return
					}
					for _, record := range recordsResp.ResourceRecordSets {
						payloadMutex.Lock()
						payload[sessionWg.AccountId][*zone.Name] = append(payload[sessionWg.AccountId][*zone.Name], record)
						payloadMutex.Unlock()
					}
				}
			}
		}()
	}

	waitGroup.Wait()

	return payload, nil
}

func AssumeRoleForAccounts(ctx context.Context, accounts []string, roleName string) (map[string]AwsSession, error) {
	payload := make(map[string]AwsSession)
	var maxConcurrentOps int64 = 30
	var waitGroup sync.WaitGroup
	payloadMutex := &sync.Mutex{}
	sem := semaphore.NewWeighted(maxConcurrentOps)

	cfg, err := awsConfig.LoadDefaultConfig(ctx, awsConfig.WithRegion("eu-west-1"), awsConfig.WithHTTPClient(CreateHttpClientWithoutKeepAlive()))
	if err != nil {
		return payload, err
	}

	for _, acc := range accounts {
		waitGroup.Add(1)
		accWg := acc
		go func() {
			sem.Acquire(ctx, 1)
			defer sem.Release(1)
			defer waitGroup.Done()

			roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accWg, roleName)

			stsClient := sts.NewFromConfig(cfg)
			roleSessionName := "oops"
			assumedRole, err := stsClient.AssumeRole(context.TODO(), &sts.AssumeRoleInput{RoleArn: &roleArn, RoleSessionName: &roleSessionName})
			if err != nil {
				util.Logger.Debug(fmt.Sprintf("unable to assume role %s, skipping account", roleArn), zap.Error(err))
				return
			}

			assumedCfg, err := awsConfig.LoadDefaultConfig(context.TODO(), awsConfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(*assumedRole.Credentials.AccessKeyId, *assumedRole.Credentials.SecretAccessKey, *assumedRole.Credentials.SessionToken)), awsConfig.WithRegion("eu-west-1"))
			if err != nil {
				util.Logger.Error(fmt.Sprintf("unable to load SDK config, %v", err))
				return
			}

			payloadMutex.Lock()
			payload[accWg] = AwsSession{
				AccountId:     accWg,
				SessionConfig: assumedCfg,
			}
			payloadMutex.Unlock()
		}()
	}

	waitGroup.Wait()

	return payload, nil
}

type AwsSession struct {
	AccountId     string
	SessionConfig aws.Config
}

// CreateHttpClientWithoutKeepAlive Currently the AWS SDK seems to let connections live for way too long. On OSes that has a very low file descriptior limit this becomes an issue.
func CreateHttpClientWithoutKeepAlive() *awsHttp.BuildableClient {
	client := awsHttp.NewBuildableClient().WithTransportOptions(func(transport *http.Transport) {
		transport.DisableKeepAlives = true
	})

	return client
}
