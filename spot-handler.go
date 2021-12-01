package main

import (
	"fmt"
	"os"
	"strconv"

	"Shubhamnegi/spot-handler-service/notification"
	"os/exec"
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

func (n Notice) terminateInstance() error {
	Logger.Info("Terminating instance %s", n.GetInstanceId())
	return TerminateInstance(n.GetInstanceId())
}

func (n Notice) ExecuteDrain(instance *ec2.Instance) {
	// var wg sync.WaitGroup
	hostname := GetHostnameByInstance(instance)
	name := GetTagNameByInstance(instance)
	asg := GetASGByInstance(instance)

	Logger.Info(fmt.Sprintf(
		"Executing node drain for instance %s hostname %s on request id %s for action %s Tag name: %s ASG: %s",
		n.GetInstanceId(),
		hostname,
		n.GetRequestId(),
		n.GetInstanceAction(),
		name,
		asg,
	))

	kubeConfig := "/var/lib/kubelet/kubeconfig"
	if os.Getenv("KUBECTL_CONFIG") != "" {
		kubeConfig = os.Getenv("KUBECTL_CONFIG")
	}

	// kubectl --kubeconfig /var/lib/kubelet/kubeconfig drain node_name
	command := fmt.Sprintf("sleep 2m && kubectl --kubeconfig %s  drain %s --ignore-daemonsets --delete-local-data", kubeConfig, hostname)
	// command := "echo 'done'"
	additonalArguments := os.Getenv("KUBECTL_ARGS")
	command = fmt.Sprintf("%s %s", command, additonalArguments)

	Logger.Info("executing:" + command)
	notification.Notify(fmt.Sprintf(
		"Executing Command: %s\nRequested for instance id: %s\nRequest id %s \nNode name: %s\nASG: %s",
		command,
		n.GetInstanceId(),
		n.GetRequestId(),
		name,
		asg,
	))
	cmd := exec.Command("sh", "-c", command)
	if err := cmd.Start(); err != nil {
		Logger.Error(err.Error())
		return
	}
	pid := cmd.Process.Pid

	Logger.Info("Command running with pid: " + strconv.Itoa(pid))
	// wg.Add(1)
	go func() {
		// defer wg.Done()
		err := cmd.Wait()
		if err == nil {
			Logger.Info(fmt.Sprintf("Command %s executed succesfully", command))
			if err = n.terminateInstance(); err != nil {
				Logger.Info(fmt.Sprintf("Error terminating instance %s %v",
					n.GetInstanceId(),
					err,
				))
			}
		} else {
			Logger.Error(fmt.Sprintf("Error executing command %s '%v'", command, err))
		}
	}()
	// wg.Wait()
}
