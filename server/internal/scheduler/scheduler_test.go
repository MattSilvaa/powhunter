package scheduler

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/MattSilvaa/powhunter/internal/db"
	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"
	dbmocks "github.com/MattSilvaa/powhunter/internal/db/mocks"
	notifymocks "github.com/MattSilvaa/powhunter/internal/notify/mocks"
	"github.com/MattSilvaa/powhunter/internal/weather"
	weathermocks "github.com/MattSilvaa/powhunter/internal/weather/mocks"
	"github.com/google/uuid"
	"go.uber.org/mock/gomock"
)

func TestCheckForecasts(t *testing.T) {
	// Set up mocks
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStoreService(ctrl)
	mockWeather := weathermocks.NewMockWeatherService(ctrl)
	mockNotify := notifymocks.NewMockNotificationService(ctrl)

	// Create test data
	resortUUID := uuid.New()
	resort := dbgen.Resort{
		ID:   1,
		Uuid: resortUUID,
		Name: "Test Resort",
		Latitude: sql.NullFloat64{
			Float64: 45.33,
			Valid:   true,
		},
		Longitude: sql.NullFloat64{
			Float64: -121.67,
			Valid:   true,
		},
	}

	forecastDate := time.Now().AddDate(0, 0, 2) // 2 days in the future
	snowPredictions := []weather.SnowPrediction{
		{
			Date:       forecastDate,
			SnowAmount: 8.5, // 8.5 inches of snow
		},
	}

	alert := db.AlertToSend{
		UserID:       1,
		UserEmail:    "user@example.com",
		UserPhone:    "+12025551234",
		ResortName:   "Test Resort",
		ResortUUID:   resortUUID.String(),
		SnowAmount:   9,
		ForecastDate: forecastDate,
	}

	// Set up mock expectations
	mockStore.EXPECT().ListAllResorts(gomock.Any()).Return([]dbgen.Resort{resort}, nil)
	mockWeather.EXPECT().GetSnowForecast(gomock.Any(), resort.Latitude.Float64, resort.Longitude.Float64).Return(snowPredictions, nil)
	mockStore.EXPECT().GetAlertMatches(gomock.Any(), resortUUID.String(), snowPredictions[0].Date, int32(9), int32(2)).Return([]db.AlertToSend{alert}, nil)
	mockNotify.EXPECT().SendSMS(alert.UserPhone, gomock.Any()).Return(nil)
	mockStore.EXPECT().RecordAlertSent(gomock.Any(), alert).Return(nil)

	// Create the scheduler
	scheduler := NewForecastScheduler(mockStore, mockWeather, mockNotify, 12*time.Hour)

	// Test the function
	err := scheduler.CheckForecasts()
	if err != nil {
		t.Fatalf("CheckForecasts returned an error: %v", err)
	}
}

func TestCheckForecastsWithErrors(t *testing.T) {
	// Set up mocks
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStoreService(ctrl)
	mockWeather := weathermocks.NewMockWeatherService(ctrl)
	mockNotify := notifymocks.NewMockNotificationService(ctrl)

	// Test case 1: Error listing resorts
	mockStore.EXPECT().ListAllResorts(gomock.Any()).Return(nil, errors.New("database error"))

	scheduler1 := NewForecastScheduler(mockStore, mockWeather, mockNotify, 12*time.Hour)
	err := scheduler1.CheckForecasts()
	if err == nil {
		t.Error("Expected error when ListAllResorts fails, but got nil")
	}

	// Test case 2: Error getting forecast
	resortUUID := uuid.New()
	resort := dbgen.Resort{
		ID:   1,
		Uuid: resortUUID,
		Name: "Test Resort",
		Latitude: sql.NullFloat64{
			Float64: 45.33,
			Valid:   true,
		},
		Longitude: sql.NullFloat64{
			Float64: -121.67,
			Valid:   true,
		},
	}

	mockStore.EXPECT().ListAllResorts(gomock.Any()).Return([]dbgen.Resort{resort}, nil)
	mockWeather.EXPECT().GetSnowForecast(gomock.Any(), resort.Latitude.Float64, resort.Longitude.Float64).Return(nil, errors.New("weather API error"))

	scheduler2 := NewForecastScheduler(mockStore, mockWeather, mockNotify, 12*time.Hour)
	err = scheduler2.CheckForecasts()
	if err != nil {
		t.Errorf("Expected no error when GetSnowForecast fails for a single resort, but got: %v", err)
	}

	// Test case 3: Error sending notification
	forecastDate := time.Now().AddDate(0, 0, 2) // 2 days in the future
	snowPredictions := []weather.SnowPrediction{
		{
			Date:       forecastDate,
			SnowAmount: 8.5, // 8.5 inches of snow
		},
	}

	alert := db.AlertToSend{
		UserID:       1,
		UserEmail:    "user@example.com",
		UserPhone:    "+12025551234",
		ResortName:   "Test Resort",
		ResortUUID:   resortUUID.String(),
		SnowAmount:   9,
		ForecastDate: forecastDate,
	}

	mockStore.EXPECT().ListAllResorts(gomock.Any()).Return([]dbgen.Resort{resort}, nil)
	mockWeather.EXPECT().GetSnowForecast(gomock.Any(), resort.Latitude.Float64, resort.Longitude.Float64).Return(snowPredictions, nil)
	mockStore.EXPECT().GetAlertMatches(gomock.Any(), resortUUID.String(), snowPredictions[0].Date, int32(9), int32(2)).Return([]db.AlertToSend{alert}, nil)
	mockNotify.EXPECT().SendSMS(alert.UserPhone, gomock.Any()).Return(errors.New("SMS sending error"))
	mockStore.EXPECT().RecordAlertSent(gomock.Any(), alert).Return(nil)

	scheduler3 := NewForecastScheduler(mockStore, mockWeather, mockNotify, 12*time.Hour)
	err = scheduler3.CheckForecasts()
	if err != nil {
		t.Errorf("Expected no error when SendSMS fails, but got: %v", err)
	}
}

func TestStartStop(t *testing.T) {
	// Set up mocks
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := dbmocks.NewMockStoreService(ctrl)
	mockWeather := weathermocks.NewMockWeatherService(ctrl)
	mockNotify := notifymocks.NewMockNotificationService(ctrl)

	// Create the scheduler with a short interval
	interval := 50 * time.Millisecond
	scheduler := NewForecastScheduler(mockStore, mockWeather, mockNotify, interval)

	// Mock expectations for the initial check and at least one scheduled check
	mockStore.EXPECT().ListAllResorts(gomock.Any()).Return(nil, nil).MinTimes(1).MaxTimes(3)

	// Start the scheduler
	scheduler.Start()

	// Wait for at least one scheduled check
	time.Sleep(interval * 2)

	// Stop the scheduler
	scheduler.Stop()

	// Verify that the scheduler stopped properly (this is implicit in the mock expectations)
}