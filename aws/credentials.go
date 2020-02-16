package aws

import (
	"errors"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type CredentialsProvider struct {
	*sts.Credentials
}

func (s CredentialsProvider) Retrieve() (awsv2.Credentials, error) {
	if s.Credentials == nil {
		return awsv2.Credentials{}, errors.New("sts credentials are nil")
	}

	return awsv2.Credentials{
		AccessKeyID:     awsv2.StringValue(s.AccessKeyId),
		SecretAccessKey: awsv2.StringValue(s.SecretAccessKey),
		SessionToken:    awsv2.StringValue(s.SessionToken),
		Expires:         awsv2.TimeValue(s.Expiration),
	}, nil
}
