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
	// if it is a confirmation reques the send confirmed
	if messageType == "SubscriptionConfirmation" {
		c.String(http.StatusOK, "Confirmed")
		return
	}
	defer c.Request.Body.Close()
	body, _ := ioutil.ReadAll(c.Request.Body)
	Logger.Info(string(body))

	// If an actual notification
	var message SNSMessage
	var notice Notice

	if err := json.Unmarshal(body, &message); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	notice = message.Message
	i := notice.GetInstanceId()
	hostname, err := getHostNameByInstanceId(i)
	if err != nil {
		Logger.Error(err.Error())
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	notice.ExecuteDrain(hostname)
	c.String(http.StatusOK, hostname)
}
