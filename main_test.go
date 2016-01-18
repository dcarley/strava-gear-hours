package main_test

import (
	. "github.com/dcarley/strava-gear-hours"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/strava/go.strava"
)

func TestStravaGearHours(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "StravaGearHours Suite")
}

type mockResponseTransport struct {
	http.Transport
	content    chan ([]byte)
	statusCode int
}

func (t *mockResponseTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var body []byte
	select {
	case c := <-t.content:
		body = c
	default:
		body = []byte(`[]`)
	}

	resp := &http.Response{
		Status:     http.StatusText(t.statusCode),
		StatusCode: t.statusCode,
		Body:       ioutil.NopCloser(bytes.NewBuffer(body)),
	}

	return resp, nil
}

func NewMockClient(content chan ([]byte), statusCode int) *strava.Client {
	t := &mockResponseTransport{
		content:    content,
		statusCode: statusCode,
	}

	httpClient := &http.Client{Transport: t}
	c := strava.NewClient("", httpClient)

	return c
}

// Useful for comparing objects that have been through JSON marshalling
// because uninitialised `time.Time` does not have the same `time.Location`
// as `time.Time{}`.
func ActivityNames(activities []*strava.ActivitySummary) (names []string) {
	for _, activity := range activities {
		names = append(names, activity.Name)
	}

	return
}

// Break an activity slice into a slice of JSON response pages.
func PaginateResponses(resp []*strava.ActivitySummary, pageSize int) (chan ([]byte), error) {
	total := len(resp)
	respChan := make(chan []byte, total)

	for min := 0; min < total; min += pageSize {
		max := min + pageSize
		if max >= total {
			max = total
		}

		buf, err := json.Marshal(resp[min:max])
		if err != nil {
			return respChan, err
		}

		respChan <- buf
	}

	return respChan, nil
}

var _ = Describe("main", func() {
	var activities []*strava.ActivitySummary

	BeforeEach(func() {
		activities = []*strava.ActivitySummary{
			{
				Name:      "ride 1",
				GearId:    "123",
				StartDate: time.Date(2016, time.January, 01, 12, 0, 0, 0, time.UTC),
			}, {
				Name:      "ride 2",
				GearId:    "456",
				StartDate: time.Date(2016, time.February, 01, 12, 0, 0, 0, time.UTC),
			}, {
				Name:      "ride 3",
				GearId:    "123",
				StartDate: time.Date(2016, time.January, 02, 12, 0, 0, 0, time.UTC),
			}, {
				Name:      "ride 4",
				GearId:    "456",
				StartDate: time.Date(2016, time.February, 02, 12, 0, 0, 0, time.UTC),
			}, {
				Name:      "ride 5",
				GearId:    "123",
				StartDate: time.Date(2016, time.January, 03, 12, 0, 0, 0, time.UTC),
			},
		}
	})

	Describe("GetBike", func() {
		var athlete *strava.AthleteDetailed

		BeforeEach(func() {
			athlete = &strava.AthleteDetailed{
				Bikes: []*strava.GearSummary{
					{
						Id:       "1",
						Name:     "road bike",
						Primary:  false,
						Distance: 100.00,
					}, {
						Id:       "2",
						Name:     "my best bike",
						Primary:  false,
						Distance: 200.00,
					}, {
						Id:       "3",
						Name:     "fat bike",
						Primary:  false,
						Distance: 300.00,
					},
				},
			}
		})

		It("should return matching bike from athlete", func() {
			buf, err := json.Marshal(athlete)
			Expect(err).To(BeNil())

			responses := make(chan []byte, 1)
			responses <- buf
			client := NewMockClient(responses, http.StatusOK)

			myBike := athlete.Bikes[1]
			out, err := GetBike(client, myBike.Name)
			Expect(err).To(BeNil())
			Expect(out).To(Equal(myBike))
		})

		It("should return error if unable to find matching bike", func() {
			buf, err := json.Marshal(athlete)
			Expect(err).To(BeNil())

			responses := make(chan []byte, 1)
			responses <- buf
			client := NewMockClient(responses, http.StatusOK)

			out, err := GetBike(client, "garbage")
			Expect(err).To(MatchError("bike not found: garbage"))
			Expect(out).To(BeNil())
		})

		It("should return errors from HTTP client", func() {
			client := NewMockClient(nil, http.StatusInternalServerError)

			out, err := GetBike(client, "garbage")
			Expect(err).To(MatchError("server error"))
			Expect(out).To(BeNil())
		})
	})

	Describe("GetActivities", func() {
		const pageSize = 2

		It("should paginate when total is divisible by pageSize", func() {
			in := activities[:len(activities)-1]
			Expect(len(in) % pageSize).To(Equal(0))

			responses, err := PaginateResponses(in, pageSize)
			Expect(err).To(BeNil())

			client := NewMockClient(responses, http.StatusOK)
			out, err := GetActivities(client, pageSize)
			Expect(err).To(BeNil())
			Expect(ActivityNames(out)).To(Equal(ActivityNames(in)))
		})

		It("should paginate when total is not divisible by pageSize", func() {
			in := activities
			Expect(len(in) % pageSize).To(Equal(1))

			responses, err := PaginateResponses(in, pageSize)
			Expect(err).To(BeNil())

			client := NewMockClient(responses, http.StatusOK)
			out, err := GetActivities(client, pageSize)
			Expect(err).To(BeNil())
			Expect(ActivityNames(out)).To(Equal(ActivityNames(in)))
		})

		It("should return errors from HTTP client", func() {
			client := NewMockClient(nil, http.StatusInternalServerError)
			out, err := GetActivities(client, pageSize)
			Expect(err).To(MatchError("server error"))
			Expect(out).To(BeNil())
		})
	})

	Describe("FilterActivities", func() {
		Describe("ByGear", func() {
			It("should return activities for gear ID 123", func() {
				gear := &strava.GearSummary{
					Id:   "123",
					Name: "my bike",
				}
				expected := []*strava.ActivitySummary{
					activities[0],
					activities[2],
					activities[4],
				}

				Expect(FilterActivities(activities, &ByGear{gear})).To(Equal(expected))
			})

			It("should return activities for gear ID 456", func() {
				gear := &strava.GearSummary{
					Id:   "456",
					Name: "my bike",
				}
				expected := []*strava.ActivitySummary{
					activities[1],
					activities[3],
				}

				Expect(FilterActivities(activities, &ByGear{gear})).To(Equal(expected))
			})

			It("should return no activities for gear ID 789", func() {
				gear := &strava.GearSummary{
					Id:   "789",
					Name: "my bike",
				}
				expected := []*strava.ActivitySummary{}

				Expect(FilterActivities(activities, &ByGear{gear})).To(Equal(expected))
			})
		})

		Describe("ByDate", func() {
			It("should return all activities", func() {
				since := time.Time{}
				expected := activities

				Expect(since.IsZero()).To(Equal(true))
				Expect(FilterActivities(activities, &ByDate{since})).To(Equal(expected))
			})

			It("should return activities since Jan 2nd", func() {
				since := time.Date(2016, time.January, 02, 0, 0, 0, 0, time.UTC)
				expected := []*strava.ActivitySummary{
					activities[1],
					activities[2],
					activities[3],
					activities[4],
				}

				Expect(FilterActivities(activities, &ByDate{since})).To(Equal(expected))
			})

			It("should return activities since Feb 1st", func() {
				since := time.Date(2016, time.February, 01, 0, 0, 0, 0, time.UTC)
				expected := []*strava.ActivitySummary{
					activities[1],
					activities[3],
				}

				Expect(FilterActivities(activities, &ByDate{since})).To(Equal(expected))
			})
		})
	})
})
