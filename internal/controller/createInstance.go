package controller

import (
	"context"
	"fmt"
	webappv1 "github.com/anhour-xyz/kubernetes-ec2-automator/api/v1"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"time"
)

func createEC2Instance(
	ctx context.Context,
	ec2Instance *webappv1.EC2Instance,
	region string,
) (string, error) {
	l := log.FromContext(ctx).WithName("createEc2Instance")
	l.Info("=== STARTING EC2 INSTANCE CREATION ===",
		"ami", ec2Instance.Spec.AmiID,
		"instanceType", ec2Instance.Spec.InstanceType,
		"region", region)

	ec2Client, err := awsClient(ctx, region)
	if err != nil {
		return "", err
	}
	runInput := &ec2.RunInstancesInput{
		ImageId:      aws.String(ec2Instance.Spec.AmiID),
		InstanceType: ec2types.InstanceType(ec2Instance.Spec.InstanceType),
		KeyName:      aws.String(ec2Instance.Spec.SshKey),
		SubnetId:     aws.String(ec2Instance.Spec.Subnet),
		MinCount:     aws.Int32(1),
		MaxCount:     aws.Int32(1),
	}

	result, err := ec2Client.RunInstances(ctx, runInput)
	if err != nil {
		return "", fmt.Errorf("create EC2 instance: %w", err)
	}

	if len(result.Instances) == 0 ||
		result.Instances[0].InstanceId == nil {
		return "", fmt.Errorf("AWS returned no EC2 instance ID")
	}

	instanceID := *result.Instances[0].InstanceId

	waiter := ec2.NewInstanceRunningWaiter(ec2Client)
	err = waiter.Wait(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}, 3*time.Minute)
	if err != nil {
		return "", fmt.Errorf(
			"wait for EC2 instance %s: %w",
			instanceID,
			err,
		)
	}

	l.Info("EC2 instance is running", "instanceID", instanceID)
	return instanceID, nil
}

func derefString(s *string) string {
	if s != nil {
		return *s
	}
	return "<nil>"
}
