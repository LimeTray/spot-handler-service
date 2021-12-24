package main

import (
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
)

type SNSMessage struct {
	Type             string    `json:"Type"`
	MessageID        string    `json:"MessageId"`
	TopicArn         string    `json:"TopicArn"`
	Subject          string    `json:"Subject"`
	Message          Notice    `json:"Message"`
	Timestamp        time.Time `json:"Timestamp"`
	SignatureVersion string    `json:"SignatureVersion"`
	Signature        string    `json:"Signature"`
	SigningCertURL   string    `json:"SigningCertURL"`
	UnsubscribeURL   string    `json:"UnsubscribeURL"`
}

type Notice struct {
	Version    string    `json:"version"`
	ID         string    `json:"id"`
	DetailType string    `json:"detail-type"`
	Source     string    `json:"source"`
	Account    string    `json:"account"`
	Time       time.Time `json:"time"`
	Region     string    `json:"region"`
	Resources  []string  `json:"resources"`
	Detail     struct {
		InstanceID     string `json:"instance-id"`
		InstanceAction string `json:"instance-action"`
	} `json:"detail"`
}

func (n Notice) GetRequestId() string {
	return n.ID
}
func (n Notice) GetInstanceId() string {
	return strings.Trim(n.Detail.InstanceID, " ")
}

func (n Notice) GetInstanceAction() string {
	return n.Detail.InstanceAction
}

func (n Notice) ExecuteDrain(instance *ec2.Instance) {
	// delayed deletion
	kh := &KubeHelper{
		instance:  instance,
		action:    n.GetInstanceAction(),
		requestId: n.GetRequestId(),
	}
	cluster := GetClusterByInstance(instance)
	if cluster == "kube.limetraydev.com" {
		defer kh.DelayedTerminateInstance()
	}
	kh.ExecuteDrain(nil) // Terminate and continue event is not required
}
