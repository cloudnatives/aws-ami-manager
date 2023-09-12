package aws

import (
	"errors"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	stsTypes "github.com/aws/aws-sdk-go-v2/service/sts/types"
)

type CredentialsProvider struct {
	*stsTypes.Credentials
}

func (s CredentialsProvider) Retrieve() (awsv2.Credentials, error) {
	if s.Credentials == nil {
		return awsv2.Credentials{}, errors.New("sts credentials are nil")
	}

	return awsv2.Credentials{
		AccessKeyID:     *s.AccessKeyId,
		SecretAccessKey: *s.SecretAccessKey,
		SessionToken:    *s.SessionToken,
		Expires:         *s.Expiration,
	}, nil
}
