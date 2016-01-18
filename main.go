package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/strava/go.strava"
)

const (
	tokenEnvVar     = "STRAVA_ACCESS_TOKEN"
	initialPage     = 1
	DefaultPageSize = 100
)

func main() {
	bikeName := flag.String("bike", "default", "bike name")
	flag.Parse()

	token := os.Getenv(tokenEnvVar)
	if token == "" {
		log.Fatalln("Environment variable not set:", tokenEnvVar)
	}
	client := strava.NewClient(token)

	bike, err := GetBike(client, *bikeName)
	if err != nil {
		log.Fatalln("Error getting bike:", err)
	}
	log.Println("Bike:", bike)

	activities, err := GetActivities(client, DefaultPageSize)
	if err != nil {
		log.Fatalln("Error getting activities:", err)
	}
	fmt.Println("Activities:", activities)
}

// GetBike retrieves a bike by name for the currently logged in user.
// Returns an error if the bike can't be found.
func GetBike(client *strava.Client, bikeName string) (*strava.GearSummary, error) {
	service := strava.NewCurrentAthleteService(client)
	athlete, err := service.Get().Do()
	if err != nil {
		return nil, err
	}

	for _, bike := range athlete.Bikes {
		if bike.Name == bikeName {
			return bike, nil
		}
	}

	return nil, fmt.Errorf("bike not found: %s", bikeName)
}

// GetActivities retrieves all activities for the currently logged in user.
func GetActivities(client *strava.Client, pageSize int) (activities []*strava.ActivitySummary, err error) {
	service := strava.NewCurrentAthleteService(client)

	for page := initialPage; ; page++ {
		pageActivities, err := service.ListActivities().Page(page).PerPage(pageSize).Do()
		if err != nil {
			return nil, err
		}

		activities = append(activities, pageActivities...)
		if len(pageActivities) < pageSize {
			break
		}
	}

	return activities, err
}
