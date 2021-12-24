package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/gin-gonic/gin"
)

// To check health of application
func healthCtrl(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

// To check health of application
func spotNoticeCtrl(c *gin.Context) {
	messageType := c.Request.Header.Get("x-amz-sns-message-type")
	Logger.Info(fmt.Sprintf("incomming request with messageType: %s", messageType))

	defer c.Request.Body.Close()
	body, _ := ioutil.ReadAll(c.Request.Body)
	Logger.Info(string(body))

	// if it is a confirmation reques the send confirmed
	if messageType == "SubscriptionConfirmation" {
		c.String(http.StatusOK, "Confirmed")
		return
	}

	var message Notice

	if err := json.Unmarshal(body, &message); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	i := message.GetInstanceId()
	instance, err := getEC2MetaByInstanceId(i)
	if err != nil {
		Logger.Error(err.Error())
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	message.ExecuteDrain(instance)
	c.String(http.StatusOK, "Request submitted")
}

func handleLifecycleEventctrl(c *gin.Context) {
	messageType := c.Request.Header.Get("x-amz-sns-message-type")
	Logger.Info(fmt.Sprintf("incomming request with messageType: %s", messageType))

	defer c.Request.Body.Close()
	body, _ := ioutil.ReadAll(c.Request.Body)
	Logger.Info(string(body))

	// if it is a confirmation reques the send confirmed
	if messageType == "SubscriptionConfirmation" {
		c.String(http.StatusOK, "Confirmed")
		return
	}

	var message LifecycleEvent
	if err := json.Unmarshal(body, &message); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if message.GetLifecycleTransition() == "autoscaling:EC2_INSTANCE_TERMINATING" {
		err := handleTermination(message)
		if err != nil {
			Logger.Error(err.Error())
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
	} else {
		msg := fmt.Sprintf("%s event is not handled", message.GetLifecycleTransition())
		Logger.Error(msg)
	}
	c.String(http.StatusOK, "Request submitted")
}

func handleTermination(message LifecycleEvent) error {
	i := message.GetEC2InstanceID()
	instance, err := getEC2MetaByInstanceId(i)
	if err != nil {
		return err
	}
	message.ExecuteDrain(instance)
	return nil
}
