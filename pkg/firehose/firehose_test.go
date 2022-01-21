package firehose

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	fluentclient "github.com/IBM/fluent-forward-go/fluent/client"
	"github.com/IBM/fluent-forward-go/fluent/client/clientfakes"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/testcontainers/testcontainers-go"
)

const (
	testToken = "testToken"
)

var (
	factory              *clientfakes.FakeConnectionFactory
	validCloudFrontEvent = &firehoseRequestBody{
		RequestID: "ed4acda5-034f-9f42-bba1-f29aea6d7d8f",
		Timestamp: 1111111,
		Records: []firehoseRecord{
			{
				Data: []byte("MTYwNzM3NDMyMS41NDEJMTI3LjAuMC4xCTAuMDQyCTIwMAk0ODUJR0VUCWh0dHAJdGVzdC5jbG91ZGZyb250Lm5ldAkvaT9zdG09MTYwNzM3NDMyMTU2MyZlPXBwJnVybD1odHRwJTI1M0ElMjUyRiUyNTJGbG9jYWxob3N0JTI1M0E1MDAwJTI1MkZwYWdlLTImcmVmcj1odHRwJTI1M0ElMjUyRiUyNTJGbG9jYWxob3N0JTI1M0E1MDAwJTI1MkYmcHBfbWl4PTAmcHBfbWF4PTAmcHBfbWl5PTAmcHBfbWF5PTAmdHY9anMtMi42LjImdG5hPWNmJmFpZD1zaXRlJnA9d2ViJnR6PUFtZXJpY2ElMjUyRk5ld19Zb3JrJmxhbmc9ZW4tVVMmY3M9VVRGLTgmcmVzPTM4NDB4MTYwMCZjZD0yNCZjb29raWU9MSZlaWQ9ZDc3ODEyN2QtNGRkZi00YzA0LTkwYWYtZmZjY2M5ODBlZWU4JmR0bT0xNjA3Mzc0MzIxNTYxJnZwPTI0NTB4MTQzMSZkcz0yNDUweDE0MzEmdmlkPTUmc2lkPWE4OGVjNzgyLTcxM2ItNGUwZC1iMmRhLWM0MDhlNTczMDgzNCZkdWlkPWVhYTY2NGY1LThiYTktNDFlOS05Yzk4LWEyYWQwODhjYTQ0MCZmcD0yMDMzMTMwOTA4CTc0NQlFV1I1Mi1DNAk2UGZaZTBjY19BalhVakZ1R25MOXBHT21GZFV4OHhSOFpVOG5yNDRKWUpXaS1EYWVKamN4a3c9PQl0ZXN0LmNsb3VkZnJvbnQubmV0CTAuMDQyCUhUVFAvMS4xCUlQdjQJTW96aWxsYS81LjAlMjAoTWFjaW50b3NoOyUyMEludGVsJTIwTWFjJTIwT1MlMjBYJTIwMTAuMTU7JTIwcnY6ODMuMCklMjBHZWNrby8yMDEwMDEwMSUyMEZpcmVmb3gvODMuMAlodHRwOi8vbG9jYWxob3N0OjUwMDAvcGFnZS0yCS0Jc3RtPTE2MDczNzQzMjE1NjMmZT1wcCZ1cmw9aHR0cCUyNTNBJTI1MkYlMjUyRmxvY2FsaG9zdCUyNTNBNTAwMCUyNTJGcGFnZS0yJnJlZnI9aHR0cCUyNTNBJTI1MkYlMjUyRmxvY2FsaG9zdCUyNTNBNTAwMCUyNTJGJnBwX21peD0wJnBwX21heD0wJnBwX21peT0wJnBwX21heT0wJnR2PWpzLTIuNi4yJnRuYT1jZiZhaWQ9c2l0ZSZwPXdlYiZ0ej1BbWVyaWNhJTI1MkZOZXdfWW9yayZsYW5nPWVuLVVTJmNzPVVURi04JnJlcz0zODQweDE2MDAmY2Q9MjQmY29va2llPTEmZWlkPWQ3NzgxMjdkLTRkZGYtNGMwNC05MGFmLWZmY2NjOTgwZWVlOCZkdG09MTYwNzM3NDMyMTU2MSZ2cD0yNDUweDE0MzEmZHM9MjQ1MHgxNDMxJnZpZD01JnNpZD1hODhlYzc4Mi03MTNiLTRlMGQtYjJkYS1jNDA4ZTU3MzA4MzQmZHVpZD1lYWE2NjRmNS04YmE5LTQxZTktOWM5OC1hMmFkMDg4Y2E0NDAmZnA9MjAzMzEzMDkwOAlNaXNzCS0JLQktCU1pc3MJLQktCWltYWdlL2dpZgkzNQktCS0JNDkzMjMJTWlzcwlVUwlnemlwLCUyMGRlZmxhdGUJaW1hZ2Uvd2VicCwqLyoJKglIb3N0OnRlc3QuY2xvdWRmcm9udC5uZXQlMEFVc2VyLUFnZW50Ok1vemlsbGEvNS4wJTIwKE1hY2ludG9zaDslMjBJbnRlbCUyME1hYyUyME9TJTIwWCUyMDEwLjE1OyUyMHJ2OjgzLjApJTIwR2Vja28vMjAxMDAxMDElMjBGaXJlZm94LzgzLjAlMEFBY2NlcHQ6aW1hZ2Uvd2VicCwqLyolMEFBY2NlcHQtTGFuZ3VhZ2U6ZW4tVVMsZW47cT0wLjUlMEFBY2NlcHQtRW5jb2Rpbmc6Z3ppcCwlMjBkZWZsYXRlJTBBRE5UOjElMEFDb25uZWN0aW9uOmtlZXAtYWxpdmUlMEFSZWZlcmVyOmh0dHA6Ly9sb2NhbGhvc3Q6NTAwMC9wYWdlLTIlMEEJSG9zdCUwQVVzZXItQWdlbnQlMEFBY2NlcHQlMEFBY2NlcHQtTGFuZ3VhZ2UlMEFBY2NlcHQtRW5jb2RpbmclMEFETlQlMEFDb25uZWN0aW9uJTBBUmVmZXJlciUwQQk4Cg=="),
			},
		},
	}
	validCloudwatchLogsEvent = &firehoseRequestBody{
		RequestID: "ed1d787c-b9e2-4631-92dc-8e7c9d26d804",
		Timestamp: 1111111,
		Records: []firehoseRecord{
			{
				Data: []byte("H4sIAMeba18AA52TX2/aMBTF3/spUJ4h/h/beUMqYy+TKsGexlSFcGm9JXFqO2Vd1e8+O7AiTUNMy0Ok3HNybN+f7+vNZJK14H31AOuXHrJykt3O1/P7T4vVar5cZNNksIcOXJKwJFpozqQg7Cg19mHp7NAnFX2LQYAC+PAuroKDqk3queyHra+d6YOx3QfTBHA+Gr5EKYq30Wa6KmlZrHz9HbR4hi6cfa/jO0pml8KZKBQrhMJKF4QLRTllBeZMc60YLbBkSlOqlBBEx0dIRaVQHI8bGnOCiW0IVZtOQgqMCcGi0Jjpd8epTWm51022fYkH2mQlLaTC0022qwKkjFjaZISjFfSIYopLQkouSk4mM8wx3mTR+2h9OPqEzAnDOSVFTjQbxRbCo92N8t3n9VjqnQ22ts1Y/Lhe3yGSH5Mc7MGBG4XHEHpfInQ4HPLema42fdXUzno/65sq7K1rc2NRW7nvEDwatuZpMMEO/pT0NMBpWwh+9LAzAVBtu2dwD9DVMLq8HVwN9yFeldHpw850RyVUIUWVDJP4OXhwM7OLzMzenDY422Rv2djNt+k1iEITxTSJHYs4C0q14EwRzNLtw4oUklKhcYRcSHYVIidXIBIpsfxviFjniuSU85wK+ifD5eISQ3qB4QmhiZ33IUIz3sdhmMWJCaaumsSQciTRs3Whav5Cz0cXoP3Q1WmKqib+Bx7ZOG+t+fnPHAWmFzjuATp4IRKrM9A0qjdvN78A1L2XllAEAAA="),
			},
		},
	}
)

func init() {
	//log.SetLevel(log.DebugLevel)
}

func TestFirehoseHandler(t *testing.T) {
	accessKey = testToken
	Convey("Given the firehose handler is invoked", t, func() {
		factory = &clientfakes.FakeConnectionFactory{}
		forwardClient = &fluentclient.Client{
			ConnectionFactory: factory,
			Timeout:           2 * time.Second,
		}
		Convey("When called without a token", func() {
			r, err := http.NewRequest("POST", "", bytes.NewBuffer([]byte(`{"commonAttributes":{"X-EVENT-TYPE":"test"}}`)))
			So(err, ShouldBeNil)
			w := httptest.NewRecorder()
			firehoseHandler(w, r)
			Convey("Then the response status code should be 401", func() {
				So(w.Code, ShouldEqual, http.StatusUnauthorized)
			})
		})

		Convey("When called with a invalid token", func() {
			r, err := http.NewRequest("POST", "", bytes.NewBuffer([]byte(`{"commonAttributes":{"X-EVENT-TYPE":"test"}}`)))
			So(err, ShouldBeNil)
			r.Header.Set(accessKeyHeaderName, testToken+"invalid")
			w := httptest.NewRecorder()
			firehoseHandler(w, r)
			Convey("Then the response status code should be 401", func() {
				So(w.Code, ShouldEqual, http.StatusUnauthorized)
			})
		})
		Convey("When called with an invalid request method", func() {
			r, err := http.NewRequest("GET", "", nil)
			So(err, ShouldBeNil)
			r.Header.Set(accessKeyHeaderName, testToken)
			w := httptest.NewRecorder()
			firehoseHandler(w, r)
			Convey("Then the response status code should be 400", func() {
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})

		Convey("When called with an empty request body", func() {
			r, err := http.NewRequest("POST", "", bytes.NewBuffer([]byte("")))
			So(err, ShouldBeNil)
			r.Header.Set(accessKeyHeaderName, testToken)
			w := httptest.NewRecorder()
			firehoseHandler(w, r)
			Convey("Then the response status code should be 400", func() {
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})

		})

		Convey("When called with a nil request body", func() {
			r, err := http.NewRequest("POST", "", nil)
			if err != nil {
				t.Fatal(err)
			}
			r.Header.Set(accessKeyHeaderName, testToken)
			w := httptest.NewRecorder()
			firehoseHandler(w, r)
			Convey("Then the response status code should be 400", func() {
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})

		})
		Convey("When called without a request ID", func() {
			r, err := http.NewRequest("POST", "", bytes.NewBuffer(nil))
			So(err, ShouldBeNil)

			r.Header.Set(accessKeyHeaderName, testToken)
			w := httptest.NewRecorder()
			firehoseHandler(w, r)
			Convey("Then the response status code should be 400", func() {
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})

		})
		Convey("When called with a valid cloudfront request", func() {
			body, _ := json.Marshal(validCloudFrontEvent)
			r, err := http.NewRequest("POST", "", bytes.NewBuffer(body))
			So(err, ShouldBeNil)
			r.Header.Set(accessKeyHeaderName, testToken)
			r.Header.Set(requestIDHeaderName, validCloudFrontEvent.RequestID)
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set(commonAttributesHeaderName, `{"commonAttributes":{"X-EVENT-TYPE":"cloudfront"}}`)
			w := httptest.NewRecorder()
			firehoseHandler(w, r)
			Convey("Then the response status code should be 200", func() {
				So(w.Code, ShouldEqual, http.StatusOK)
			})
		})

		Convey("When called with a valid cloudwatch logs request", func() {
			body, _ := json.Marshal(validCloudwatchLogsEvent)
			r, err := http.NewRequest("POST", "", bytes.NewBuffer(body))
			So(err, ShouldBeNil)
			r.Header.Set(accessKeyHeaderName, testToken)
			r.Header.Set(accessKeyHeaderName, testToken)
			r.Header.Set(requestIDHeaderName, validCloudFrontEvent.RequestID)
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set(commonAttributesHeaderName, `{"commonAttributes":{"X-EVENT-TYPE":"cloudwatchlogs"}}`)
			w := httptest.NewRecorder()
			firehoseHandler(w, r)
			Convey("Then the response status code should be 200", func() {
				So(w.Code, ShouldEqual, http.StatusOK)
			})
		})
	})
}

func TestParseEventType(t *testing.T) {
	tt := []struct {
		name                string
		commonAttributes    string
		eventTypeHeaderName string
		want                string
		wantErr             bool
	}{
		{
			name:                "valid event type",
			commonAttributes:    `{"commonAttributes":{"X-EVENT-TYPE":"test"}}`,
			want:                "test",
			eventTypeHeaderName: "X-EVENT-TYPE",
			wantErr:             false,
		},
		{
			name:                "no event type",
			commonAttributes:    `{"commonAttributes":{"X-INVALID-TYPE":"cloudfront"}}`,
			want:                "unknown",
			eventTypeHeaderName: "X-EVENT-TYPE",
			wantErr:             true,
		},
	}

	for _, tc := range tt {
		Convey(fmt.Sprintf("When event type is %s", tc.want), t, func() {
			eventTypeHeaderName = tc.eventTypeHeaderName
			r, err := http.NewRequest("", "", nil)
			So(err, ShouldBeNil)
			r.Header.Set(commonAttributesHeaderName, tc.commonAttributes)
			So(parseEventType(r), ShouldEqual, tc.want)
		})

	}
}

func TestFluentbitMessage(t *testing.T) {
	accessKey = testToken
	eventTypeHeaderName = "X-EVENT-TYPE"
	forwardClient = &fluentclient.Client{
		ConnectionFactory: &fluentclient.TCPConnectionFactory{
			Target: fluentclient.ServerAddress{
				Hostname: "localhost",
				Port:     24224,
			},
		},
	}
	Convey("Given fluentbit is running", t, func() {
		provider, _ := testcontainers.NewDockerProvider()
		req := testcontainers.ContainerRequest{
			Image:        "testfluentbit:latest",
			ExposedPorts: []string{"0.0.0.0:24224:24224/tcp"},
			Cmd:          []string{"/fluent-bit/bin/fluent-bit", "-c", "/fluent-bit/etc/fluent-bit.conf"},
			SkipReaper:   true,
			//WaitingFor:   wait.For
		}
		c, err := provider.RunContainer(context.Background(), req)
		So(err, ShouldBeNil)
		defer c.Terminate(context.Background())
		Convey("It should output a cloudfront log", func() {
			//logrus.SetLevel(logrus.DebugLevel)
			err := forwardClient.Connect()
			So(err, ShouldBeNil)
			body, _ := json.Marshal(validCloudFrontEvent)
			r, err := http.NewRequest("POST", "", bytes.NewBuffer(body))
			So(err, ShouldBeNil)
			r.Header.Set(accessKeyHeaderName, testToken)
			r.Header.Set(requestIDHeaderName, validCloudFrontEvent.RequestID)
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set(commonAttributesHeaderName, `{"commonAttributes":{"X-EVENT-TYPE":"cloudfront"}}`)
			w := httptest.NewRecorder()
			firehoseHandler(w, r)
			Convey("Then the response status code should be 200", func() {
				So(w.Code, ShouldEqual, http.StatusOK)
			})
			Convey("Then the fluentbit log should contain the cloudfront event", func() {
				time.Sleep(5 * time.Second) // wait for fluentbit to write the log
				logs, err := c.Logs(context.Background())
				So(err, ShouldBeNil)
				b, err := ioutil.ReadAll(logs)
				So(err, ShouldBeNil)
				So(string(b), ShouldContainSubstring, `"type":"cloudfront"`)

			})
		})
		Convey("It should output cloudwatch logs", func() {
			//logrus.SetLevel(logrus.DebugLevel)
			err := forwardClient.Connect()
			So(err, ShouldBeNil)
			body, _ := json.Marshal(validCloudwatchLogsEvent)
			r, err := http.NewRequest("POST", "", bytes.NewBuffer(body))
			So(err, ShouldBeNil)
			r.Header.Set(accessKeyHeaderName, testToken)
			r.Header.Set(requestIDHeaderName, validCloudFrontEvent.RequestID)
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set(commonAttributesHeaderName, `{"commonAttributes":{"X-EVENT-TYPE":"cloudwatchlogs"}}`)
			w := httptest.NewRecorder()
			firehoseHandler(w, r)
			Convey("Then the response status code should be 200", func() {
				So(w.Code, ShouldEqual, http.StatusOK)
			})
			Convey("Then the fluentbit log should contain the cloudwatch logs event", func() {
				time.Sleep(5 * time.Second) // wait for fluentbit to write the log
				logs, err := c.Logs(context.Background())
				So(err, ShouldBeNil)
				b, err := ioutil.ReadAll(logs)
				So(err, ShouldBeNil)
				So(string(b), ShouldContainSubstring, `"type":"cloudwatchlogs"`)
				So(string(b), ShouldContainSubstring, `"owner":"071959437513"`)
				So(string(b), ShouldContainSubstring, `"logGroupName":"/jesse/test"`)
				So(string(b), ShouldContainSubstring, `"user-identifier":"feeney1708"`)
			})
		})
	})
}
