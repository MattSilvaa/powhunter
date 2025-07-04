// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0

package db

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type AlertHistory struct {
	ID           int32         `json:"id"`
	UserUuid     uuid.NullUUID `json:"user_uuid"`
	ResortUuid   uuid.NullUUID `json:"resort_uuid"`
	SentAt       sql.NullTime  `json:"sent_at"`
	ForecastDate time.Time     `json:"forecast_date"`
	SnowAmount   float64       `json:"snow_amount"`
}

type Resort struct {
	ID          int32           `json:"id"`
	Uuid        uuid.UUID       `json:"uuid"`
	Name        string          `json:"name"`
	UrlHost     sql.NullString  `json:"url_host"`
	UrlPathname sql.NullString  `json:"url_pathname"`
	Latitude    sql.NullFloat64 `json:"latitude"`
	Longitude   sql.NullFloat64 `json:"longitude"`
}

type User struct {
	ID        int32          `json:"id"`
	Uuid      uuid.UUID      `json:"uuid"`
	Email     string         `json:"email"`
	Phone     sql.NullString `json:"phone"`
	CreatedAt sql.NullTime   `json:"created_at"`
}

type UserAlert struct {
	ID               int32         `json:"id"`
	UserUuid         uuid.NullUUID `json:"user_uuid"`
	ResortUuid       uuid.NullUUID `json:"resort_uuid"`
	MinSnowAmount    float64       `json:"min_snow_amount"`
	NotificationDays int32         `json:"notification_days"`
	Active           sql.NullBool  `json:"active"`
	CreatedAt        sql.NullTime  `json:"created_at"`
}
