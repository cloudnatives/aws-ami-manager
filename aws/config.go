package aws

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"os"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
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

	role string
}

func NewConfigurationManager() *ConfigurationManager {
	return NewConfigurationManagerForRegionsAndAccounts(make([]string, 0), make([]string, 0), "")
}

func NewConfigurationManagerForRegionsAndAccounts(regions []string, accounts []string, role string) *ConfigurationManager {
	cm := &ConfigurationManager{
		regions:  regions,
		accounts: accounts,
		role:     role,
	}

	log.Debug("Setting defaults")
	conf, err := config.LoadDefaultConfig(context.TODO())

	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}

	cm.defaultConfig = conf
	cm.defaultProfile = os.Getenv(ProfileString)
	cm.defaultRegion = conf.Region

	stsService := sts.NewFromConfig(conf)

	defaultAccountID, err := stsService.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		panic("unable to load defaultAccountId, " + err.Error())
	}

	cm.defaultAccountID = defaultAccountID.Account

	cm.configsPerAccount = make(map[string]awsv2.Config)
	for _, account := range cm.accounts {
		// you shouldn't assume role in your own account. We expect this user to have sufficient permissions
		if account == *cm.defaultAccountID {
			continue
		}

		confCopy := conf.Copy()

		confCopy.Credentials = stscreds.NewAssumeRoleProvider(stsService, "arn:aws:iam::"+account+":role/"+cm.role)

		cm.configsPerAccount[account] = confCopy
	}

	return cm
}

func (cm *ConfigurationManager) GetDefaultRegion() string {
	return cm.defaultRegion
}

func (cm *ConfigurationManager) GetDefaultAccountID() *string {
	return cm.defaultAccountID
}

func (cm *ConfigurationManager) loadConfiguration() {
	log.Debug("Load configuration")

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
	conf := cm.getConfigurationForAccount(account)
	conf.Region = region

	return conf
}

func (cm *ConfigurationManager) getAccounts() []string {
	return cm.accounts
}
