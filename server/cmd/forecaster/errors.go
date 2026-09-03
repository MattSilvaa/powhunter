package main

import "errors"

var (
	errMissingTwilioConfig = errors.New("twilio credentials are not configured")
	errRunIncomplete       = errors.New("forecast run did not complete for every resort")
)
