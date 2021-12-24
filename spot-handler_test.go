package main

import (
	// "fmt"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthCheck(t *testing.T) {
	rr := httptest.NewRecorder()

	r := server()
	routes(r)
	request, err := http.NewRequest("GET", "/health", nil)
	assert.NoError(t, err)

	r.ServeHTTP(rr, request)
	assert.Equal(t, 200, rr.Code)
}

func TestSpotHandler(t *testing.T) {
	registerLogger()
	ec2auth()

	r := server()

	routes(r)
	t.Run("Check path exist", func(t *testing.T) {
		rr := httptest.NewRecorder()

		body := strings.NewReader("Test body")
		request, err := http.NewRequest("POST", "/api/v1/notice", body)
		request.Header.Set("x-amz-sns-message-type", "SubscriptionConfirmation")
		assert.NoError(t, err)

		r.ServeHTTP(rr, request)
		assert.NotEqual(t, 404, rr.Code)
	})

	t.Run("Accepts confirmation request", func(t *testing.T) {
		rr := httptest.NewRecorder()

		body := strings.NewReader("Test body")
		request, err := http.NewRequest("POST", "/api/v1/notice", body)
		request.Header.Set("x-amz-sns-message-type", "SubscriptionConfirmation")
		assert.NoError(t, err)

		r.ServeHTTP(rr, request)

		assert.Equal(t, rr.Body.String(), "Confirmed")
		assert.Equal(t, 200, rr.Code)
	})

	t.Run("Process Notice", func(t *testing.T) {
		rr := httptest.NewRecorder()

		body := strings.NewReader("{\n  \"version\": \"0\",\n  \"id\": \"9266de61-51b0-fc39-708c-375a6d3a1f8c\",\n  \"detail-type\": \"EC2 Spot Instance Interruption Warning\",\n  \"source\": \"aws.ec2\",\n  \"account\": \"445897275450\",\n  \"time\": \"2021-11-23T00:48:56Z\",\n  \"region\": \"ap-southeast-1\",\n  \"resources\": [\"arn:aws:ec2:ap-southeast-1a:instance/i-0125e73c3ad672c07\"],\n  \"detail\": {\n    \"instance-id\": \"i-0125e73c3ad672c07\"}\n}")
		request, err := http.NewRequest("POST", "/api/v1/notice", body)
		request.Header.Set("x-amz-sns-message-type", "Notification")
		assert.NoError(t, err)

		r.ServeHTTP(rr, request)
		fmt.Println("Body: " + rr.Body.String())

		assert.NotEmpty(t, rr.Body.String())
		assert.Equal(t, 200, rr.Code)
	})
	t.Run("Process LifecycleEvent", func(t *testing.T) {
		rr := httptest.NewRecorder()

		body := strings.NewReader("{ \"version\": \"0\", \"id\": \"468fe059-f4b7-445f-bb22-2a271b94974d\", \"detail-type\": \"EC2 Instance-terminate Lifecycle Action\", \"source\": \"aws.autoscaling\", \"account\": \"123456789012\", \"time\": \"2015-12-22T18:43:48Z\", \"region\": \"us-east-1\", \"resources\": [\"arn:aws:autoscaling:us-east-1:123456789012:autoScalingGroup:59fcbb81-bd02-485d-80ce-563ef5b237bf:autoScalingGroupName/sampleASG\"], \"detail\": { \"LifecycleActionToken\": \"630aa23f-48eb-45e7-aba6-799ea6093a0f\", \"AutoScalingGroupName\": \"sampleASG\", \"LifecycleHookName\": \"SampleLifecycleHook-6789\", \"EC2InstanceId\": \"i-0ac76206687372e93\", \"LifecycleTransition\": \"autoscaling:EC2_INSTANCE_TERMINATING\" } }")
		request, err := http.NewRequest("POST", "/api/v1/lifecycle-event", body)
		request.Header.Set("x-amz-sns-message-type", "Notification")
		assert.NoError(t, err)

		r.ServeHTTP(rr, request)
		fmt.Println("Body: " + rr.Body.String())

		assert.NotEmpty(t, rr.Body.String())
		assert.Equal(t, 200, rr.Code)
	})

}
