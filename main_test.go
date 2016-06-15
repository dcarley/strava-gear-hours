package main_test

import (
	. "github.com/dcarley/strava-gear-hours"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/strava/go.strava"
)

func TestStravaGearHours(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "StravaGearHours Suite")
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

var _ = Describe("main", func() {
	var (
		server     *ghttp.Server
		client     *strava.Client
		activities []*strava.ActivitySummary
	)

	BeforeEach(func() {
		server = ghttp.NewServer()
		serverURL, err := url.Parse(server.URL())
		Expect(err).To(BeNil())

		mockDial := func(network, addr string) (net.Conn, error) {
			return net.Dial(network, serverURL.Host)
		}

		httpClient := &http.Client{
			Transport: &http.Transport{
				Dial:    mockDial,
				DialTLS: mockDial,
			},
		}
		client = strava.NewClient("", httpClient)

		activities = []*strava.ActivitySummary{
			{
				Name:       "ride 1",
				GearId:     "123",
				StartDate:  time.Date(2016, time.January, 01, 12, 0, 0, 0, time.UTC),
				MovingTime: int(time.Hour.Seconds()),
			}, {
				Name:       "ride 2",
				GearId:     "456",
				StartDate:  time.Date(2016, time.February, 01, 12, 0, 0, 0, time.UTC),
				MovingTime: 0,
			}, {
				Name:       "ride 3",
				GearId:     "123",
				StartDate:  time.Date(2016, time.January, 02, 12, 0, 0, 0, time.UTC),
				MovingTime: int(2 * time.Hour.Seconds()),
			}, {
				Name:       "ride 4",
				GearId:     "456",
				StartDate:  time.Date(2016, time.February, 02, 12, 0, 0, 0, time.UTC),
				MovingTime: int(45 * time.Minute.Seconds()),
			}, {
				Name:       "ride 5",
				GearId:     "123",
				StartDate:  time.Date(2016, time.January, 03, 12, 0, 0, 0, time.UTC),
				MovingTime: int(30 * time.Second.Seconds()),
			},
		}
	})

	Describe("GetBike", func() {
		Context("good response from API", func() {
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

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v3/athlete"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, athlete),
					),
				)
			})

			It("should return matching bike from athlete", func() {
				myBike := athlete.Bikes[1]
				out, err := GetBike(client, myBike.Name)
				Expect(err).To(BeNil())
				Expect(out).To(Equal(myBike))
			})

			It("should return error if unable to find matching bike", func() {
				out, err := GetBike(client, "garbage")
				Expect(err).To(MatchError("bike not found: garbage"))
				Expect(out).To(BeNil())
			})
		})

		Context("bad response from API", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v3/athlete"),
						ghttp.RespondWith(http.StatusInternalServerError, "error"),
					),
				)
			})

			It("should return errors from HTTP client", func() {
				out, err := GetBike(client, "garbage")
				Expect(err).To(MatchError("server error"))
				Expect(out).To(BeNil())
			})
		})
	})

	Describe("GetActivities", func() {
		const pageSize = 2

		Context("total is divisible by pageSize", func() {
			BeforeEach(func() {
				activities = activities[:len(activities)-1]
				Expect(len(activities) % pageSize).To(Equal(0))

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v3/athlete/activities"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, activities[0:2]),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v3/athlete/activities"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, activities[2:4]),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v3/athlete/activities"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, []*strava.ActivitySummary{}),
					),
				)
			})

			It("should paginate through all results", func() {
				out, err := GetActivities(client, pageSize)
				Expect(err).To(BeNil())
				Expect(ActivityNames(out)).To(Equal(ActivityNames(activities)))
			})
		})

		Context("total is not divisible by pageSize", func() {
			BeforeEach(func() {
				Expect(len(activities) % pageSize).To(Equal(1))

				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v3/athlete/activities"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, activities[0:2]),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v3/athlete/activities"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, activities[2:4]),
					),
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v3/athlete/activities"),
						ghttp.RespondWithJSONEncoded(http.StatusOK, activities[4:5]),
					),
				)
			})

			It("should paginate through all results", func() {
				out, err := GetActivities(client, pageSize)
				Expect(err).To(BeNil())
				Expect(ActivityNames(out)).To(Equal(ActivityNames(activities)))
			})
		})

		Context("bad response from API", func() {
			BeforeEach(func() {
				server.AppendHandlers(
					ghttp.CombineHandlers(
						ghttp.VerifyRequest("GET", "/api/v3/athlete/activities"),
						ghttp.RespondWith(http.StatusInternalServerError, "error"),
					),
				)
			})

			It("should return errors from HTTP client", func() {
				out, err := GetActivities(client, pageSize)
				Expect(err).To(MatchError("server error"))
				Expect(out).To(BeNil())
			})
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

	Describe("SumMovingTime", func() {
		It("should sum moving time from all activities", func() {
			expected, err := time.ParseDuration("3h45m30s")
			Expect(err).To(BeNil())
			Expect(SumMovingTime(activities)).To(Equal(expected))
		})
	})
})
