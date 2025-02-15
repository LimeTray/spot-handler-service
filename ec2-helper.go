package main

import (
	"Shubhamnegi/spot-handler-service/notification"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var AWS_Session *session.Session

func ec2auth() {
	// WILL BE USING ENV FOR AWS CREDENTAILS
	// AWS_ACCESS_KEY_ID
	// AWS_SECRET_ACCESS_KEY

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("AWS_REGION"))},
	)
	if err != nil {
		panic(err)
	}
	AWS_Session = sess
}

func getEC2MetaByInstanceId(instanceId string) (*ec2.Instance, error) {
	svc := ec2.New(AWS_Session)
	input := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceId),
		},
	}
	result, err := svc.DescribeInstances(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				Logger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			Logger.Error(aerr.Error())
		}
		return nil, err
	}
	if len(result.Reservations) == 0 || len(result.Reservations[0].Instances) == 0 {
		return nil, errors.New("invalid instance")
	}
	return result.Reservations[0].Instances[0], nil
}

func GetHostNameByInstanceId(instanceId string) (string, error) {
	Logger.Info("Getting instance details for " + instanceId)
	if i, err := getEC2MetaByInstanceId(instanceId); err != nil {
		return "", err
	} else {
		return *i.PrivateDnsName, nil
	}
}

func GetHostnameByInstance(instance *ec2.Instance) string {
	return *instance.PrivateDnsName
}

func getTagValueByInstace(instance *ec2.Instance, key string) string {
	val := ""
	tags := instance.Tags
	if len(tags) > 0 {
		for _, t := range tags {
			if strings.EqualFold(strings.ToLower(*t.Key), strings.ToLower(key)) {
				val = *t.Value
				break
			}
		}
	}
	return val
}

func GetTagNameByInstance(instance *ec2.Instance) string {
	return getTagValueByInstace(instance, "name")
}

func GetASGByInstance(instance *ec2.Instance) string {
	return getTagValueByInstace(instance, "aws:autoscaling:groupName")
}

func GetClusterByInstance(instance *ec2.Instance) string {
	return getTagValueByInstace(instance, "KubernetesCluster")
}

func TerminateInstance(instanceId string, sleepInterval time.Duration) error {
	time.Sleep(sleepInterval)
	notification.Notify(fmt.Sprintf(
		"Terminating instance %s",
		instanceId,
	))
	svc := ec2.New(AWS_Session)
	input := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceId),
		},
	}
	if _, err := svc.TerminateInstances(input); err != nil {
		return err
	} else {
		return nil
	}
}

func SendContinueEvent(lifecycleEvent *LifecycleEvent) {
	notification.Notify(fmt.Sprintf(
		"sending continue event for instanceId %s",
		lifecycleEvent.GetEC2InstanceID(),
	))

	svc := autoscaling.New(AWS_Session)
	if _, err := svc.CompleteLifecycleAction(&autoscaling.CompleteLifecycleActionInput{
		LifecycleHookName:     aws.String(lifecycleEvent.GetLifecycleHookName()),
		AutoScalingGroupName:  aws.String(lifecycleEvent.GetAutoScalingGroupName()),
		LifecycleActionToken:  aws.String(lifecycleEvent.GetLifecycleActionToken()),
		LifecycleActionResult: aws.String("CONTINUE"),
	}); err != nil {
		Logger.Error(fmt.Sprintf("Error sending continue event %s %v",
			lifecycleEvent.GetEC2InstanceID(),
			err,
		))
	} else {
		Logger.Info("Continue event sent for instnace id: " + lifecycleEvent.GetEC2InstanceID())
	}
}
