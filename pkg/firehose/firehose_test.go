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
			name:   "valid request",
			method: "POST",
			want:   "ed4acda5-034f-9f42-bba1-f29aea6d7d8f",
			payload: &firehoseRequestBody{
				RequestID: "ed4acda5-034f-9f42-bba1-f29aea6d7d8f",
				Timestamp: 1111111,
				Records: []firehoseRecord{
					{
						Data: []byte("eyJoZWxsbyI6ICJ3b3JsZCJ9Cg=="),
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
