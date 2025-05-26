package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/MattSilvaa/powhunter/internal/db"
	dbgen "github.com/MattSilvaa/powhunter/internal/db/generated"

	"github.com/google/uuid"
)

// Resort represents a ski resort from the JSON file
type Resort struct {
	Name string `json:"name"`
	URL  struct {
		Host     string `json:"host"`
		PathName string `json:"pathname"`
	} `json:"url"`
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func main() {
	data, err := os.ReadFile("cmd/seed/data/resorts.json")
	if err != nil {
		log.Fatalf("Failed to read resorts data: %v", err)
	}

	var resorts []Resort
	if err := json.Unmarshal(data, &resorts); err != nil {
		log.Fatalf("Failed to parse resorts data: %v", err)
	}

	dbConn, err := db.New()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	queries := dbgen.New(dbConn)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err = queries.ClearResorts(ctx)
	if err != nil {
		log.Fatalf("Failed to clear resorts: %v", err)
	}

	for _, r := range resorts {
		_, err := queries.InsertResort(ctx, dbgen.InsertResortParams{
			Uuid: uuid.New(),
			Name: r.Name,
			UrlHost: sql.NullString{
				String: r.URL.Host,
				Valid:  r.URL.Host != "",
			},
			UrlPathname: sql.NullString{
				String: r.URL.PathName,
				Valid:  r.URL.PathName != "",
			},
			Latitude: sql.NullFloat64{
				Float64: r.Lat,
				Valid:   true,
			},
			Longitude: sql.NullFloat64{
				Float64: r.Lon,
				Valid:   true,
			},
		})
		if err != nil {
			log.Fatalf("Failed to insert resort %s: %v", r.Name, err)
		}
		fmt.Printf("Inserted resort: %s\n", r.Name)
	}

	fmt.Println("Resort seeding completed successfully")
}
