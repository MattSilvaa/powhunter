package forecast_test

import (
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/MattSilvaa/powhunter/internal/db"
	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	dbmocks "github.com/MattSilvaa/powhunter/internal/db/mocks"
	"github.com/MattSilvaa/powhunter/internal/forecast"
	notifymocks "github.com/MattSilvaa/powhunter/internal/notify/mocks"
	"github.com/MattSilvaa/powhunter/internal/weather"
	weathermocks "github.com/MattSilvaa/powhunter/internal/weather/mocks"
)

const resortUUIDText = "11111111-1111-4111-8111-111111111111"

func fixedNow() time.Time {
	return time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
}

func forecastDate() time.Time {
	return time.Date(2026, 1, 12, 0, 0, 0, 0, time.UTC)
}

func resortUUID() uuid.UUID {
	return uuid.MustParse(resortUUIDText)
}

type harness struct {
	store    *dbmocks.MockStoreService
	weather  *weathermocks.MockWeatherService
	notifier *notifymocks.MockNotificationService
	runner   *forecast.Runner
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	ctrl := gomock.NewController(t)
	store := dbmocks.NewMockStoreService(ctrl)
	weatherClient := weathermocks.NewMockWeatherService(ctrl)
	notifier := notifymocks.NewMockNotificationService(ctrl)

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	return &harness{
		store:    store,
		weather:  weatherClient,
		notifier: notifier,
		runner: forecast.NewRunner(store, weatherClient, notifier, logger,
			fixedNow,
			forecast.Options{Concurrency: 1, Attempts: 2}),
	}
}

func testResort() dbgen.Resort {
	return dbgen.Resort{
		Uuid:      resortUUID(),
		Name:      "Vail",
		Latitude:  sql.NullFloat64{Float64: 39.6403, Valid: true},
		Longitude: sql.NullFloat64{Float64: -106.3742, Valid: true},
	}
}

func testAlert() db.AlertToSend {
	return db.AlertToSend{
		UserUuid:     uuid.New(),
		UserEmail:    "skier@example.com",
		UserPhone:    "+16195733405",
		ResortName:   "Vail",
		ResortUUID:   resortUUID(),
		SnowAmount:   8,
		ForecastDate: forecastDate(),
	}
}

func (h *harness) expectOneMatchingAlert(alert db.AlertToSend) {
	h.store.EXPECT().ListAllResorts(gomock.Any()).Return([]dbgen.Resort{testResort()}, nil)
	h.weather.EXPECT().
		GetSnowForecast(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]weather.WeatherPrediction{{Date: forecastDate(), SnowAmount: 8}}, nil)
	h.store.EXPECT().
		GetAlertMatches(gomock.Any(), resortUUID().String(), forecastDate(), 8.0, int32(2)).
		Return([]db.AlertToSend{alert}, nil)
}

func TestASuccessfulSendIsRecordedExactlyOnce(t *testing.T) {
	h := newHarness(t)
	alert := testAlert()

	h.expectOneMatchingAlert(alert)
	h.notifier.EXPECT().SendSMS(alert.UserPhone, gomock.Any()).Return(nil).Times(1)
	h.store.EXPECT().RecordAlertSent(gomock.Any(), alert).Return(nil).Times(1)

	run, err := h.runner.Run(t.Context())

	require.NoError(t, err)
	assert.Equal(t, 1, run.AlertsSent)
	assert.Zero(t, run.AlertsFailed)
	assert.True(t, run.Succeeded)
}

// A failed send used to be recorded anyway, which permanently suppressed that
// storm for the user.
func TestAFailedSendIsNotRecorded(t *testing.T) {
	h := newHarness(t)
	alert := testAlert()

	h.expectOneMatchingAlert(alert)
	h.notifier.EXPECT().SendSMS(alert.UserPhone, gomock.Any()).Return(errors.New("twilio down"))
	h.store.EXPECT().RecordAlertSent(gomock.Any(), gomock.Any()).Times(0)

	run, err := h.runner.Run(t.Context())

	require.NoError(t, err)
	assert.Zero(t, run.AlertsSent)
	assert.Equal(t, 1, run.AlertsFailed)
}

// Likewise, a user with no phone number had history written for an alert that
// was never attempted.
func TestAUserWithoutAPhoneNumberIsNeitherSentNorRecorded(t *testing.T) {
	h := newHarness(t)
	alert := testAlert()
	alert.UserPhone = ""

	h.expectOneMatchingAlert(alert)
	h.notifier.EXPECT().SendSMS(gomock.Any(), gomock.Any()).Times(0)
	h.store.EXPECT().RecordAlertSent(gomock.Any(), gomock.Any()).Times(0)

	run, err := h.runner.Run(t.Context())

	require.NoError(t, err)
	assert.Zero(t, run.AlertsSent)
}

func TestADeliveredAlertWithAFailedRecordIsReported(t *testing.T) {
	h := newHarness(t)
	alert := testAlert()

	h.expectOneMatchingAlert(alert)
	h.notifier.EXPECT().SendSMS(alert.UserPhone, gomock.Any()).Return(nil)
	h.store.EXPECT().RecordAlertSent(gomock.Any(), alert).Return(errors.New("write failed"))

	run, err := h.runner.Run(t.Context())

	require.NoError(t, err)
	assert.Equal(t, 1, run.AlertsSent)
	assert.Equal(t, 1, run.RecordsFailed, "a duplicate on the next run must be visible")
}

func TestATransientForecastFailureIsRetried(t *testing.T) {
	h := newHarness(t)

	h.store.EXPECT().ListAllResorts(gomock.Any()).Return([]dbgen.Resort{testResort()}, nil)

	gomock.InOrder(
		h.weather.EXPECT().
			GetSnowForecast(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, errors.New("temporary failure")),
		h.weather.EXPECT().
			GetSnowForecast(gomock.Any(), gomock.Any(), gomock.Any()).
			Return([]weather.WeatherPrediction{{Date: forecastDate(), SnowAmount: 8}}, nil),
	)

	h.store.EXPECT().
		GetAlertMatches(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, nil)

	run, err := h.runner.Run(t.Context())

	require.NoError(t, err)
	assert.Zero(t, run.ResortsFailed)
	assert.True(t, run.Succeeded)
}

// One resort failing must not stop the others, and the run must not claim
// success, which is how a broken pipeline stayed invisible.
func TestOneFailingResortDoesNotStopTheOthers(t *testing.T) {
	h := newHarness(t)

	failing := testResort()
	failing.Uuid = uuid.MustParse("22222222-2222-4222-8222-222222222222")
	failing.Name = "Broken"

	h.store.EXPECT().ListAllResorts(gomock.Any()).Return([]dbgen.Resort{failing, testResort()}, nil)

	h.weather.EXPECT().
		GetSnowForecast(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("upstream down")).Times(2)
	h.weather.EXPECT().
		GetSnowForecast(gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]weather.WeatherPrediction{{Date: forecastDate(), SnowAmount: 8}}, nil)

	h.store.EXPECT().
		GetAlertMatches(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, nil)

	run, err := h.runner.Run(t.Context())

	require.NoError(t, err)
	assert.Equal(t, 2, run.ResortsChecked)
	assert.Equal(t, 1, run.ResortsFailed)
	assert.False(t, run.Succeeded, "a run that could not reach a resort is not a success")
}

func TestResortsWithoutCoordinatesAreSkipped(t *testing.T) {
	h := newHarness(t)

	incomplete := testResort()
	incomplete.Latitude = sql.NullFloat64{}

	h.store.EXPECT().ListAllResorts(gomock.Any()).Return([]dbgen.Resort{incomplete}, nil)
	h.weather.EXPECT().GetSnowForecast(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	run, err := h.runner.Run(t.Context())

	require.NoError(t, err)
	assert.Zero(t, run.ResortsChecked)
}
