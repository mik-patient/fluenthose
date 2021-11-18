package firehose

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/IBM/fluent-forward-go/fluent/client"
	"github.com/IBM/fluent-forward-go/fluent/client/clientfakes"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

const (
	testToken = "testToken"
)

var (
	factory    *clientfakes.FakeConnectionFactory
	clientSide net.Conn
)

func TestFirehoseHandler(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	factory = &clientfakes.FakeConnectionFactory{}
	forwardClient = &Client{
		ConnectionFactory: factory,
		Timeout:           2 * time.Second,
	}
	accessKey = testToken
	tt := []struct {
		name       string
		method     string
		payload    *firehoseRequestBody
		want       string
		statusCode int
		token      string
	}{
		{
			name:   "invalid request method",
			method: "GET",
			payload: &firehoseRequestBody{
				RequestID: "testRequestID",
			},
			want:       "bad request",
			statusCode: http.StatusBadRequest,
			token:      testToken,
		},
		{
			name:       "empty request body",
			method:     "POST",
			want:       "bad request",
			payload:    &firehoseRequestBody{},
			statusCode: http.StatusBadRequest,
			token:      testToken,
		},
		{
			name:       "nil request body",
			method:     "POST",
			want:       "bad request",
			payload:    nil,
			statusCode: http.StatusBadRequest,
			token:      testToken,
		},
		{
			name:   "missing token",
			method: "POST",
			want:   "unauthorized",
			payload: &firehoseRequestBody{
				RequestID: "testRequestID",
			},
			statusCode: http.StatusUnauthorized,
			token:      "",
		},
		{
			name:   "valid cloudfront request",
			method: "POST",
			want:   "ed4acda5-034f-9f42-bba1-f29aea6d7d8f",
			payload: &firehoseRequestBody{
				RequestID: "ed4acda5-034f-9f42-bba1-f29aea6d7d8f",
				Timestamp: 1111111,
				Records: []firehoseRecord{
					{
						Data: []byte("MTYzNzE4OTY5OC41NDAgIDguOC44LjggMC4wMDAgICA0MDMgICAgIDEyNjUgICAgR0VUICAgICBodHRwcyAgIHRlc3QuZXhhbXBsZS5jb20gICAgL3BhdGggICAgIDQ4OCAgICAgVFhMNTAtUDIgOVYwcVJvLTFKOG1yWG1fRnpwYll6TmdGMHdzWTVFQjhmdThVLXV1cU81WHFRPT0geHh4eHh4eHguY2xvdWRmcm9udC5uZXQgICAwLjAwMCAgIEhUVFAvMS4xICAgICAgICBJUHY0ICAgIEdvLWh0dHAtY2xpZW50LzEuMSAgICAgIC0tLSAgICAgICBFcnJvciAgIC0gICAgICAgVExTdjEuMyBUTFNfQUVTXzEyOF9HQ01fU0hBMjU2ICBFcnJvciAgIC0gICAgICAgLSAgICAgICB0ZXh0L2h0bWwgICAgICAgOTE5ICAgICAtICAgICAgIC0gICAgICAgMzMwMDggICBFcnJvciAgIFVTICAgICAgZ3ppcCAgICAgIHRleHQvaHRtbCxhcHBsaWNhdGlvbi94aHRtbCt4bWwsYXBwbGljYXRpb24veG1sO3E9MC45LGltYWdlL3dlYnAsaW1hZ2UvYXBuZywqLyo7cT0wLjgsYXBwbGljYXRpb24vc2lnbmVkLWV4Y2hhbmdlO3Y9YjM7cT0wLjkgICAgKiAgICAgICBIb3N0Omhvc3QuZXhhbXBsZS5jb20lMEFVc2VyLUFnZW50OkdvLWh0dHAtY2xpZW50LzEuMSUwQUFjY2VwdDp0ZXh0L2h0bWwsYXBwbGljYXRpb24veGh0bWwreG1sLGFwcGxpY2F0aW9uL3htbDtxPTAuOSxpbWFnZS93ZWJwLGltYWdlL2FwbmcsKi8qO3E9MC44LGFwcGxpY2F0aW9uL3NpZ25lZC1leGNoYW5nZTt2PWIzO3E9MC45JTBBQWNjZXB0LUxhbmd1YWdlOmVuLVVTLGVuO3E9MC45LGNhO3E9MC44JTBBQ2FjaGUtQ29udHJvbDpuby1jYWNoZSUwQVByYWdtYTpuby1jYWNoZSUwQVNlYy1GZXRjaC1EZXN0OmRvY3VtZW50JTBBU2VjLUZldGNoLU1vZGU6bmF2aWdhdGUlMEFTZWMtRmV0Y2gtU2l0ZTpub25lJTBBU2VjLUZldGNoLVVzZXI6PzElMEFVcGdyYWRlLUluc2VjdXJlLVJlcXVlc3RzOjElMEFBY2NlcHQtRW5jb2Rpbmc6Z3ppcCUwQUNvbm5lY3Rpb246Y2xvc2UlMEEgICAgSG9zdCUwQVVzZXItQWdlbnQlMEFBY2NlcHQlMEFBY2NlcHQtTGFuZ3VhZ2UlMEFDYWNoZS1Db250cm9sJTBBUHJhZ21hJTBBU2VjLUZldGNoLURlc3QlMEFTZWMtRmV0Y2gtTW9kZSUwQVNlYy1GZXRjaC1TaXRlJTBBU2VjLUZldGNoLVVzZXIlMEFVcGdyYWRlLUluc2VjdXJlLVJlcXVlc3RzJTBBQWNjZXB0LUVuY29kaW5nJTBBQ29ubmVjdGlvbiUwQSAgIDEzCgo="),
					},
				},
			},
			statusCode: http.StatusOK,
			token:      testToken,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.payload)
			req, err := http.NewRequest(tc.method, "", bytes.NewBuffer(body))
			if err != nil {
				t.Fatal(err)
			}

			req.Header.Set(accessKeyHeaderName, tc.token)
			if tc.payload != nil {
				req.Header.Set(requestIDHeaderName, tc.payload.RequestID)
			}
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			firehoseHandler(w, req)

			if w.Code != tc.statusCode {
				t.Errorf("got %d, want %d", w.Code, tc.statusCode)
			}

			assert.Contains(t, w.Body.String(), tc.want)
		})
	}
}
