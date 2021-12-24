package main

import (
	"Shubhamnegi/spot-handler-service/notification"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"
)

type KubeHelper struct {
	instance  *ec2.Instance
	action    string
	requestId string
}

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

func (kh KubeHelper) TerminateInstance() error {
	instanceId := *kh.instance.InstanceId
	Logger.Info("Terminating instance %s after 30 sec", instanceId)
	return TerminateInstance(instanceId, 30*time.Second)
}

func (kh KubeHelper) DelayedTerminateInstance() error {
	instanceId := *kh.instance.InstanceId
	Logger.Info("Terminating instance %s after 10 min", instanceId)
	return TerminateInstance(instanceId, 10*time.Minute)
}

func (kh KubeHelper) DeleteNode() error {
	// kp2 delete node
	instance := kh.instance
	hostname := GetHostnameByInstance(instance)
	name := GetTagNameByInstance(instance)
	asg := GetASGByInstance(instance)

	Logger.Info(fmt.Sprintf(
		"Executing node delete for instance %s hostname %s on request id %s for action %s Tag name: %s ASG: %s",
		*instance.InstanceId,
		hostname,
		kh.requestId,
		kh.action,
		name,
		asg,
	))

	k := getKubectl()
	command := fmt.Sprintf("%s delete node %s", k, hostname)
	Logger.Info("executing:" + command)
	notification.Notify(fmt.Sprintf(
		"Executing Command: %s\nRequested for instance id: %s\nRequest id %s \nNode name: %s\nASG: %s",
		command,
		*instance.InstanceId,
		kh.requestId,
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

func (kh KubeHelper) ExecuteDrain(
	lifecycleEvent *LifecycleEvent,
) {
	// var wg sync.WaitGroup
	instance := kh.instance
	hostname := GetHostnameByInstance(instance)
	name := GetTagNameByInstance(instance)
	asg := GetASGByInstance(instance)

	Logger.Info(fmt.Sprintf(
		"Executing node drain for instance %s hostname %s on request id %s for action %s Tag name: %s ASG: %s after 2 min",
		*instance.InstanceId,
		hostname,
		kh.requestId,
		kh.action,
		name,
		asg,
	))

	k := getKubectl()
	// kubectl --kubeconfig /var/lib/kubelet/kubeconfig drain node_name
	command := fmt.Sprintf(
		"%s  drain %s --ignore-daemonsets --delete-local-data",
		k,
		hostname,
	)
	// command := "echo 'done'"
	additonalArguments := os.Getenv("KUBECTL_ARGS")
	command = fmt.Sprintf("%s %s", command, additonalArguments)

	notification.Notify(fmt.Sprintf(
		"Executing Command: %s\nRequested for instance id: %s\nRequest id %s \nNode name: %s\nASG: %s after 2 min",
		command,
		*instance.InstanceId,
		kh.requestId,
		name,
		asg,
	))
	// wg.Add(1)
	go func() {
		// defer wg.Done()
		if lifecycleEvent != nil {
			// In case of termination notification send continue irrespective of sucess/failure other wise node will get stuck in termination till heart beat time specified
			// Not requor
			defer SendContinueEvent(lifecycleEvent)
		}

		time.Sleep(2 * time.Minute)
		Logger.Info("executing:" + command)
		cmd := exec.Command("sh", "-c", command)
		if err := cmd.Start(); err != nil {
			Logger.Error(err.Error())
		}
		pid := cmd.Process.Pid

		Logger.Info("Command running with pid: " + strconv.Itoa(pid))

		err := cmd.Wait()
		if err == nil {
			Logger.Info(fmt.Sprintf("Command %s executed succesfully", command))
			if err = kh.DeleteNode(); err != nil {
				// error deleting
				Logger.Info(fmt.Sprintf("Error deleting node %s %v",
					hostname,
					err,
				))
			} else {
				// deleted
				if lifecycleEvent == nil {
					// If this is not lifecycle event then delete the node manually
					if err = kh.TerminateInstance(); err != nil {
						Logger.Info(fmt.Sprintf("Error terminating instance %s %v",
							*instance.InstanceId,
							err,
						))
					} else {
						Logger.Info(fmt.Sprintf("Terminated instance %s sucessfully", *instance.InstanceId))
					}
				}
			}
		} else {
			Logger.Error(fmt.Sprintf("Error executing command %s '%v'", command, err))
		}

	}()
	// wg.Wait()
}
