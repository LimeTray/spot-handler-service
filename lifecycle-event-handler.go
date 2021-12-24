package main

import (
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
)

type LifecycleEvent struct {
	Version    string    `json:"version"`
	ID         string    `json:"id"`
	DetailType string    `json:"detail-type"`
	Source     string    `json:"source"`
	Account    string    `json:"account"`
	Time       time.Time `json:"time"`
	Region     string    `json:"region"`
	Resources  []string  `json:"resources"`
	Detail     struct {
		LifecycleActionToken string `json:"LifecycleActionToken"`
		AutoScalingGroupName string `json:"AutoScalingGroupName"`
		LifecycleHookName    string `json:"LifecycleHookName"`
		EC2InstanceID        string `json:"EC2InstanceId"`
		LifecycleTransition  string `json:"LifecycleTransition"`
	} `json:"detail"`
}

func (event LifecycleEvent) GeID() string {
	return event.ID
}
func (event LifecycleEvent) GetLifecycleTransition() string {
	return event.Detail.LifecycleTransition
}

func (event LifecycleEvent) GetLifecycleActionToken() string {
	return event.Detail.LifecycleActionToken
}

func (event LifecycleEvent) GetLifecycleHookName() string {
	return event.Detail.LifecycleHookName
}

func (event LifecycleEvent) GetAutoScalingGroupName() string {
	return event.Detail.AutoScalingGroupName
}

func (event LifecycleEvent) GetEC2InstanceID() string {
	return event.Detail.EC2InstanceID
}

func (event LifecycleEvent) ExecuteDrain(instance *ec2.Instance) {
	kh := &KubeHelper{
		instance:  instance,
		action:    event.GetLifecycleTransition(),
		requestId: event.GeID(),
	}
	kh.ExecuteDrain(&event)
}
