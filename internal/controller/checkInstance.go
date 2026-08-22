package controller

import (
	"context"
	"errors"
	"fmt"

	webappv1 "github.com/anhour-xyz/kubernetes-ec2-automator/api/v1"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/smithy-go"
)

func checkEC2InstanceExists(ctx context.Context, instanceID string, ec2Instance *webappv1.EC2Instance) (bool, *ec2types.Instance, error) {
	if instanceID == "" {
		return false, nil, nil
	}

	ec2Client, err := awsClient(ctx, ec2Instance.Spec.Region)
	if err != nil {
		return false, nil, err
	}

	result, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID}})

	if err != nil {
		var apiErr smithy.APIError
		if errors.As(err, &apiErr) && apiErr.ErrorCode() == "InvalidInstanceID.NotFound" {
			return false, nil, nil
		}

		return false, nil, fmt.Errorf(
			"describe EC2 instance %s: %w",
			instanceID,
			err,
		)
	}

	if len(result.Reservations) == 0 {
		return false, nil, nil
	}

	if len(result.Reservations[0].Instances) == 0 {
		return false, nil, nil
	}

	instance := &result.Reservations[0].Instances[0]
	return true, instance, nil
}
