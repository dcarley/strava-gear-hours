package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/strava/go.strava"
)

const (
	dateFormat      = "2006-01-02"
	tokenEnvVar     = "STRAVA_ACCESS_TOKEN"
	initialPage     = 1
	DefaultPageSize = 100
)

func main() {
	bikeName := flag.String("bike", "default", "bike name")
	sinceStr := flag.String("since", "1970-01-01",
		fmt.Sprintf("since date, in format %q", dateFormat),
	)
	flag.Parse()

	since, err := time.Parse(dateFormat, *sinceStr)
	if err != nil {
		log.Fatalln("Error parsing date:", err)
	}

	token := os.Getenv(tokenEnvVar)
	if token == "" {
		log.Fatalln("Environment variable not set:", tokenEnvVar)
	}
	client := strava.NewClient(token)

	bike, err := GetBike(client, *bikeName)
	if err != nil {
		log.Fatalln("Error getting bike:", err)
	}

	activities, err := GetActivities(client, DefaultPageSize)
	if err != nil {
		log.Fatalln("Error getting activities:", err)
	}

	activities = FilterActivities(activities, &ByGear{bike})
	activities = FilterActivities(activities, &ByDate{since})
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

type ActivityFilterer interface {
	Select(*strava.ActivitySummary) bool
}

type ByGear struct {
	*strava.GearSummary
}

func (b *ByGear) Select(activity *strava.ActivitySummary) bool {
	return activity.GearId != b.Id
}

type ByDate struct {
	time.Time
}

func (b *ByDate) Select(activity *strava.ActivitySummary) bool {
	return activity.StartDate.Before(b.UTC())
}

// FilterActivities returns a slice of `activities` that satisfy `filter`.
func FilterActivities(activities []*strava.ActivitySummary, filter ActivityFilterer) []*strava.ActivitySummary {
	for i := 0; i < len(activities); i++ {
		if filter.Select(activities[i]) {
			activities = append(activities[:i], activities[i+1:]...)
			i--
		}
	}

	return activities
}
