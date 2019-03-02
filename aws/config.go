package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"os"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	log "github.com/sirupsen/logrus"
)

const (
	ProfileString  string = "AWS_PROFILE"
	defaultProfile string = "default"
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

	stsService *sts.STS
}

func NewConfigurationManager(regions []string, accounts []string) *ConfigurationManager {
	cm := &ConfigurationManager{
		regions:  regions,
		accounts: accounts,
	}

	cm.setDefaults()
	cm.loadConfiguration()

	return cm
}

func (cm *ConfigurationManager) getSTSService() *sts.STS {
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
	output, err := cm.getSTSService().GetCallerIdentityRequest(&sts.GetCallerIdentityInput{}).Send()

	if err != nil {
		log.Fatal(err)
	}
	return output.Account
}

func (cm *ConfigurationManager) GetDefaultRegion() *string {
	return &cm.defaultRegion
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

	svc := cm.getSTSService()

	for _, account := range cm.accounts {
		// you cannot assume role in your own account
		if account == *cm.defaultAccountID {
			continue
		}
		input := &sts.AssumeRoleInput{
			RoleArn:         awsv2.String("arn:aws:iam::" + account + ":role/OrganizationAccountAccessRole"),
			RoleSessionName: awsv2.String("cli"),
		}

		output, err := svc.AssumeRoleRequest(input).Send()

		if err != nil {
			log.Fatal(err)
		}

		awsConfig := svc.Config.Copy()
		awsConfig.Credentials = CredentialsProvider{Credentials: output.Credentials}

		cm.configsPerAccount[account] = awsConfig
	}
}

func (cm *ConfigurationManager) GetConfigurationForDefaultAccount() awsv2.Config {
	return cm.getConfigurationForAccount(*cm.defaultAccountID)
}

func (cm *ConfigurationManager) getConfigurationForAccount(account string) awsv2.Config {
	if account == *cm.defaultAccountID {
		return cm.defaultConfig
	}
	return cm.configsPerAccount[account]
}

func (cm *ConfigurationManager) getConfigurationForDefaultAccountAndRegion(region string) awsv2.Config {
	config := cm.GetConfigurationForDefaultAccount()
	config.Region = region

	return config
}

func (cm *ConfigurationManager) getConfigurationForAccountAndRegion(account string, region string) awsv2.Config {
	config := cm.getConfigurationForAccount(account)
	config.Region = region

	return config
}

func (cm *ConfigurationManager) getAccounts() []string {
	return cm.accounts
}
