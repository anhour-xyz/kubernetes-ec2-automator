package controller

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func awsClient(
	ctx context.Context,
	region string,
) (*ec2.Client, error) {
	if region == "" {
		return nil, fmt.Errorf("AWS region is required")
	}

	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("load AWS configuration: %w", err)
	}

	return ec2.NewFromConfig(cfg), nil
}
