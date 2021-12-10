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

func getkubeConfig() string {
	kubeConfig := "/var/lib/kubelet/kubeconfig"
	if os.Getenv("KUBECTL_CONFIG") != "" {
		kubeConfig = os.Getenv("KUBECTL_CONFIG")
	}
	return kubeConfig
}

func getKubectl() string {
	config := getkubeConfig()
	kt := fmt.Sprintf("kubectl --kubeconfig %s", config)
	return kt
}

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
	time.Sleep(30 * time.Second)
	Logger.Info("Terminating instance %s", n.GetInstanceId())
	return TerminateInstance(n.GetInstanceId())
}

func (n Notice) DeleteNode(instance *ec2.Instance) error {
	// kp2 delete node
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
	k := getKubectl()
	command := fmt.Sprintf("%s delete node %s", k, hostname)
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
		return err
	}
	if err := cmd.Wait(); err != nil {
		Logger.Error(err.Error())
		return err
	} else {
		Logger.Info(fmt.Sprintf("Command %s executed sucessfully", command))
	}

	return nil
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

	k := getKubectl()
	// kubectl --kubeconfig /var/lib/kubelet/kubeconfig drain node_name
	command := fmt.Sprintf(
		"sleep 2m &&  %s  drain %s --ignore-daemonsets --delete-local-data",
		k,
		hostname,
	)
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
			if err = n.DeleteNode(instance); err != nil {
				// error deleting
				Logger.Info(fmt.Sprintf("Error deleting node %s %v",
					hostname,
					err,
				))
			} else {
				// deleted
				if err = n.terminateInstance(); err != nil {
					Logger.Info(fmt.Sprintf("Error terminating instance %s %v",
						n.GetInstanceId(),
						err,
					))
				} else {
					Logger.Info(fmt.Sprintf("Terminated instance %s sucessfully", n.GetInstanceId()))
				}
			}
		} else {
			Logger.Error(fmt.Sprintf("Error executing command %s '%v'", command, err))
		}
	}()
	// wg.Wait()
}
