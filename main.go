package main

import (
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
	token := os.Getenv(tokenEnvVar)
	if token == "" {
		log.Fatalln("Environment variable not set:", tokenEnvVar)
	}
	client := strava.NewClient(token)

	activities, err := GetActivities(client, DefaultPageSize)
	if err != nil {
		log.Fatalln("Error getting activities:", err)
	}
	fmt.Println("Activities:", activities)
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
