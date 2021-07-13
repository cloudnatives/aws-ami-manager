package aws

import (
	"context"
	"os"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	log "github.com/sirupsen/logrus"
)

const (
	ProfileString string = "AWS_PROFILE"
)

func SetLogLevel(level string) {
	logLevel, err := log.ParseLevel(level)

	if err != nil {
		log.Fatal("Invalid loglevel: %s", level)
	}

	log.SetLevel(logLevel)
}

type ConfigurationManager struct {
	defaultConfig    awsv2.Config
	defaultRegion    string
	defaultProfile   string
	defaultAccountID *string

	regions  []string
	accounts []string

	configsPerAccount map[string]awsv2.Config

	stsService *sts.Client

	role string
}

func NewConfigurationManager() *ConfigurationManager {
	cm := &ConfigurationManager{}
	cm.setDefaults()

	return cm
}

func NewConfigurationManagerForRegionsAndAccounts(regions []string, accounts []string, role string) *ConfigurationManager {
	cm := &ConfigurationManager{
		regions:  regions,
		accounts: accounts,
		role: role,
	}

	cm.setDefaults()
	cm.loadConfiguration()

	return cm
}

func (cm *ConfigurationManager) getSTSClient() *sts.Client {
	if cm.stsService == nil {
		cfg, err := external.LoadDefaultAWSConfig()

		if err != nil {
			log.Fatal(err)
		}
		cm.stsService = sts.New(cfg)
	}
	return cm.stsService
}

func (cm *ConfigurationManager) GetAccountID() *string {
	output, err := cm.getSTSClient().GetCallerIdentityRequest(&sts.GetCallerIdentityInput{}).Send(context.Background())

	if err != nil {
		log.Fatal(err)
	}
	return output.Account
}

func (cm *ConfigurationManager) GetDefaultRegion() string {
	return cm.defaultRegion
}

func (cm *ConfigurationManager) GetDefaultAccountID() *string {
	return cm.defaultAccountID
}

func (cm *ConfigurationManager) setDefaults() {
	log.Debug("Setting defaults")
	config, err := external.LoadDefaultAWSConfig()

	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}

	cm.defaultConfig = config
	cm.defaultProfile = os.Getenv(ProfileString)
	cm.defaultRegion = config.Region
	cm.defaultAccountID = cm.GetAccountID()

	cm.configsPerAccount = make(map[string]awsv2.Config)
}

func (cm *ConfigurationManager) loadConfiguration() {
	log.Debug("Load configuration")

	svc := cm.getSTSClient()

	for _, account := range cm.accounts {
		// you shouldn't assume role in your own account. We expect this user to have sufficient permissions
		if account == *cm.defaultAccountID {
			continue
		}
		input := &sts.AssumeRoleInput{
			RoleArn:         awsv2.String("arn:aws:iam::" + account + ":role/" + cm.role),
			RoleSessionName: awsv2.String("cli"),
		}

		output, err := svc.AssumeRoleRequest(input).Send(context.Background())

		if err != nil {
			log.Fatal(err)
		}

		awsConfig := svc.Config.Copy()
		credentials := output.Credentials
		awsConfig.Credentials = awsv2.NewStaticCredentialsProvider(
			*credentials.AccessKeyId,
			*credentials.SecretAccessKey,
			*credentials.SessionToken,
		)

		cm.configsPerAccount[account] = awsConfig
	}
}

func (cm *ConfigurationManager) GetConfigurationForDefaultAccount() awsv2.Config {
	log.Debug("GetConfigurationForDefaultAccount")
	return cm.getConfigurationForAccount(*cm.defaultAccountID)
}

func (cm *ConfigurationManager) getConfigurationForAccount(account string) awsv2.Config {
	log.Debugf("getConfigurationForAccount: account: %s", account)
	if account == *cm.defaultAccountID {
		return cm.defaultConfig
	}
	return cm.configsPerAccount[account]
}

func (cm *ConfigurationManager) getConfigurationForDefaultAccountAndRegion(region string) awsv2.Config {
	log.Debugf("getConfigurationForDefaultAccountAndRegion: region: %s", region)
	config := cm.GetConfigurationForDefaultAccount()
	config.Region = region

	return config
}

func (cm *ConfigurationManager) getConfigurationForAccountAndRegion(account string, region string) awsv2.Config {
	log.Debugf("getConfigurationForAccountAndRegion - Account: %s, Region: %s", account, region)
	config := cm.getConfigurationForAccount(account)
	config.Region = region

	return config
}

func (cm *ConfigurationManager) getAccounts() []string {
	return cm.accounts
}
