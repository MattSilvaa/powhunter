package db_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	generated "github.com/MattSilvaa/powhunter/internal/db/generated"
	"github.com/MattSilvaa/powhunter/internal/db/mocks"
)

func TestListAllResorts(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mocks.NewMockStoreService(ctrl)

	ctx := context.Background()
	expectedResorts := []generated.Resort{
		{
			ID:          1,
			Uuid:        uuid.New(),
			Name:        "Test Resort 1",
			UrlHost:     sql.NullString{String: "resort1.com", Valid: true},
			UrlPathname: sql.NullString{String: "/snow", Valid: true},
			Latitude:    sql.NullFloat64{Float64: 40.0, Valid: true},
			Longitude:   sql.NullFloat64{Float64: -105.0, Valid: true},
		},
		{
			ID:          2,
			Uuid:        uuid.New(),
			Name:        "Test Resort 2",
			UrlHost:     sql.NullString{String: "resort2.com", Valid: true},
			UrlPathname: sql.NullString{String: "/conditions", Valid: true},
			Latitude:    sql.NullFloat64{Float64: 41.0, Valid: true},
			Longitude:   sql.NullFloat64{Float64: -106.0, Valid: true},
		},
	}

	mockStore.EXPECT().
		ListAllResorts(gomock.Any()).
		Return(expectedResorts, nil)

	resorts, err := mockStore.ListAllResorts(ctx)

	assert.NoError(t, err)
	assert.Equal(t, len(expectedResorts), len(resorts))
	assert.Equal(t, expectedResorts[0].Name, resorts[0].Name)
	assert.Equal(t, expectedResorts[0].ID, resorts[0].ID)
	assert.Equal(t, expectedResorts[0].UrlHost.String, resorts[0].UrlHost.String)
	assert.Equal(t, expectedResorts[0].UrlPathname.String, resorts[0].UrlPathname.String)
	assert.Equal(t, expectedResorts[0].Latitude.Float64, resorts[0].Latitude.Float64)
	assert.Equal(t, expectedResorts[0].Longitude.Float64, resorts[0].Longitude.Float64)
}
