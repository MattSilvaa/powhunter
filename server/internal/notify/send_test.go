package notify

import (
	"errors"
	"testing"

	twilioAPI "github.com/twilio/twilio-go/rest/api/v2010"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeMessageAPI records the parameters SendSMS hands to Twilio.
type fakeMessageAPI struct {
	params *twilioAPI.CreateMessageParams
	calls  int
	err    error
}

func (f *fakeMessageAPI) CreateMessage(
	params *twilioAPI.CreateMessageParams,
) (*twilioAPI.ApiV2010Message, error) {
	f.calls++
	f.params = params

	if f.err != nil {
		return nil, f.err
	}

	return &twilioAPI.ApiV2010Message{}, nil
}

func TestSendSMSUsesTheRequestedRecipient(t *testing.T) {
	api := &fakeMessageAPI{}
	client := &TwilioClient{fromNumber: "+15550000000", api: api}

	require.NoError(t, client.SendSMS("+15551234567", "Powder Alert!"))

	require.Equal(t, 1, api.calls)
	require.NotNil(t, api.params.To)

	// Regression: the recipient was previously hardcoded, so every subscriber's
	// alert went to one phone number.
	assert.Equal(t, "+15551234567", *api.params.To)
	assert.Equal(t, "+15550000000", *api.params.From)
	assert.Equal(t, "Powder Alert!", *api.params.Body)
}

func TestSendSMSRejectsEmptyInput(t *testing.T) {
	api := &fakeMessageAPI{}
	client := &TwilioClient{fromNumber: "+15550000000", api: api}

	assert.Error(t, client.SendSMS("", "body"))
	assert.Error(t, client.SendSMS("+15551234567", ""))
	assert.Zero(t, api.calls, "no request should reach Twilio for invalid input")
}

func TestSendSMSPropagatesTwilioErrors(t *testing.T) {
	api := &fakeMessageAPI{err: errors.New("twilio unavailable")}
	client := &TwilioClient{fromNumber: "+15550000000", api: api}

	err := client.SendSMS("+15551234567", "Powder Alert!")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "twilio unavailable")
}
