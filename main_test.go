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
				Name: "ride 1",
			}, {
				Name: "ride 2",
			}, {
				Name: "ride 3",
			}, {
				Name: "ride 4",
			}, {
				Name: "ride 5",
			},
		}
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
})
