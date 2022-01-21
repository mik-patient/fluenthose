package firehose

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/heptiolabs/healthcheck"

	"github.com/IBM/fluent-forward-go/fluent/protocol"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	muxlogrus "github.com/pytimer/mux-logrus"

	fluentclient "github.com/IBM/fluent-forward-go/fluent/client"
	log "github.com/sirupsen/logrus"
)

const (
	accessKeyHeaderName        = "X-Amz-Firehose-Access-Key"
	requestIDHeaderName        = "X-Amz-Firehose-Request-Id"
	commonAttributesHeaderName = "X-Amz-Firehose-Common-Attributes"
)

var (
	errAuth       = &firehoseAPIError{code: http.StatusUnauthorized, msg: "unauthorized"}
	errBadReq     = &firehoseAPIError{code: http.StatusBadRequest, msg: "bad request"}
	forwardClient *fluentclient.Client
	accessKey     string
	eventsTotal   = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "fluenthose_events_total",
			Help: "Number of events processed by type",
		},
		[]string{"type", "status"},
	)
	eventTypeHeaderName string
)

func init() {
	prometheus.MustRegister(eventsTotal)
}

type APIError interface {
	APIError() (int, string, string)
}

type firehoseAPIError struct {
	code      int
	msg       string
	requestID string
}

func (e firehoseAPIError) Error() string {
	return e.msg
}

func (e firehoseAPIError) APIError() (int, string, string) {
	return e.code, e.msg, e.requestID
}

// firehoseCommonAttributes represents common attributes (metadata).
type firehoseCommonAttributes struct {
	CommonAttributes map[string]string `json:"commonAttributes"`
}

// firehoseRequestBody represents request body.
type firehoseRequestBody struct {
	RequestID string           `json:"requestId,omitempty"`
	Timestamp int64            `json:"timestamp,omitempty"`
	Records   []firehoseRecord `json:"records,omitempty"`
}

// firehoseRecord represents records in request body.
type firehoseRecord struct {
	Data []byte `json:"data"`
}

// firehoseResponseBody represents response body.
type firehoseResponseBody struct {
	RequestID    string `json:"requestId,omitempty"`
	Timestamp    int64  `json:"timestamp,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// cloudwatchlogsevent represents cloudwatchlogs event.
type cloudWatchLogsEvent struct {
	Owner               string                        `json:"owner"`
	LogGroup            string                        `json:"logGroup"`
	LogStream           string                        `json:"logStream"`
	SubscriptionFilters []string                      `json:"subscriptionFilters"`
	MessageType         string                        `json:"messageType"`
	Timestamp           int64                         `json:"timestamp"`
	LogEvents           []cloudWatchLogsEventLogEvent `json:"logEvents"`
}

type cloudWatchLogsEventLogEvent struct {
	ID        string `json:"id"`
	Message   string `json:"message"`
	Timestamp int64  `json:"timestamp"`
}

func RunFirehoseServer(address, key, forwardAddress, eventTypeHeader string) {
	eventTypeHeaderName = eventTypeHeader
	accessKey = key
	forwardHost, forwardPort, err := net.SplitHostPort(forwardAddress)
	if err != nil {
		log.Fatalf("Failed to parse forward address: %s", err)
	}
	forwardPortInt, _ := strconv.Atoi(forwardPort)
	forwardClient = &fluentclient.Client{
		ConnectionFactory: &fluentclient.TCPConnectionFactory{
			Target: fluentclient.ServerAddress{
				Hostname: forwardHost,
				Port:     forwardPortInt,
			},
		},
	}
	err = forwardClient.Connect()
	if err != nil {
		log.Fatalf("error connecting to fluent forwarder: %s", err)
	}

	health := healthcheck.NewHandler()
	health.AddLivenessCheck(
		"forwarder",
		healthcheck.TCPDialCheck(forwardAddress, 50*time.Millisecond))
	health.AddReadinessCheck(
		"forwarder",
		healthcheck.TCPDialCheck(forwardAddress, 50*time.Millisecond))

	logOptions := muxlogrus.LogOptions{
		Formatter:      &log.JSONFormatter{},
		EnableStarting: true,
	}
	loggingMiddleware := muxlogrus.NewLogger(logOptions)

	router := mux.NewRouter()
	router.Handle("/", loggingMiddleware.Middleware(http.HandlerFunc(firehoseHandler))).Methods("POST")
	router.Handle("/metrics", promhttp.Handler())
	router.HandleFunc("/health/live", health.LiveEndpoint)
	router.HandleFunc("/health/ready", health.ReadyEndpoint)

	srv := &http.Server{
		Addr:    address,
		Handler: router,
	}
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	log.Infof("Fluenthose server listening on %s", address)
	log.Debugf("log-level: %s, fowarding to: %s", log.GetLevel(), forwardAddress)
	<-done

	// shutdown gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		forwardClient.Disconnect()
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	log.Infof("fluenthose Exited Properly")
}

func firehoseHandler(w http.ResponseWriter, r *http.Request) {
	log.Debugf("firehose %s request received from %s", r.Method, r.RemoteAddr)
	key := r.Header.Get(accessKeyHeaderName)
	if key == "" || key != accessKey {
		JSONHandleError(w, errAuth)
		return

	}

	requestID := r.Header.Get(requestIDHeaderName)
	if requestID == "" {
		log.Debugf("requestID header is missing")
		JSONHandleError(w, errBadReq)
		return
	}

	resp := firehoseResponseBody{
		RequestID: requestID,
	}

	log.Debugf("%s request from %s", r.Method, r.RemoteAddr)
	log.Debugf("request headers: %+v", r.Header)
	log.Debugf("body: %s", r.Body)

	eventType := parseEventType(r)
	firehoseReq, err := parseRequestBody(r)
	if err != nil {
		log.Errorf("failed to parse request body: %s", err)
		JSONHandleError(w, errBadReq)
		return
	}

	for _, record := range firehoseReq.Records {
		switch eventType {
		case "cloudwatchlogs":
			if err = forwardCloudwatchLog(record.Data, requestID); err != nil {
				log.Errorf("failed to forward cloudwatchlogs event: %s", err)
				continue
			}
		case "cloudfront":
			if err = forwardCloudfrontEvent(record.Data, requestID); err != nil {
				log.Errorf("failed to forward cloudfront event: %s", err)
				continue
			}
		default: // do nothing
		}
	}
	resp.Timestamp = time.Now().UnixNano() / int64(time.Millisecond)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func forwardCloudfrontEvent(data []byte, requestID string) error {
	var recordCount = 0
	log.Debugf("firehose record: %s", string(data))
	// decode base64 encoded data
	decodedData, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		log.Errorf("failed to decode base64 encoded data: %s", err)
		return err
	}
	log.Debugf("firehose record decoded: %s", decodedData)
	recordCount++
	msg := &protocol.Message{
		Tag:       "cloudfront",
		Timestamp: time.Now().UTC().Unix(),
		Record: map[string]interface{}{
			"data": string(decodedData),
			"type": "cloudfront",
		},
		Options: &protocol.MessageOptions{},
	}
	err = forwardClient.SendMessage(msg)
	if err != nil {
		eventsTotal.WithLabelValues("eventType", "error").Inc()
		log.Errorf("failed to send message: %s", err)
	} else {
		eventsTotal.WithLabelValues("eventType", "success").Inc()
		log.Infof("%d records sent to fluent forwarder", recordCount)

	}
	return nil
}

func forwardCloudwatchLog(data []byte, requestID string) error {
	// base64 decode and gunzip event data
	decodedData, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return err
	}
	unzippedData, err := gzip.NewReader(bytes.NewReader(decodedData))
	if err != nil {
		return err
	}
	defer unzippedData.Close()
	var logRecord cloudWatchLogsEvent
	err = json.NewDecoder(unzippedData).Decode(&logRecord)
	if err != nil {
		return err
	}
	log.Debugf("cloudwatch log record: %+v", logRecord)
	var logGroupName = logRecord.LogGroup
	var logStreamName = logRecord.LogStream
	var logEvents = logRecord.LogEvents
	for _, logEvent := range logEvents {
		msg := &protocol.Message{
			Tag:       "cloudwatchlogs",
			Timestamp: logEvent.Timestamp,
			Record: map[string]interface{}{
				"owner":         logRecord.Owner,
				"logGroupName":  logGroupName,
				"logStreamName": logStreamName,
				"message":       logEvent.Message,
				"timestamp":     logEvent.Timestamp,
				"requestID":     requestID,
				"type":          "cloudwatchlogs",
			},
			Options: &protocol.MessageOptions{},
		}
		log.Debugf("cloudwatch log message: %+v", msg)
		err := forwardClient.SendMessage(msg)
		if err != nil {
			eventsTotal.WithLabelValues("eventType", "error").Inc()
			log.Errorf("failed to send message: %s", err)
		} else {
			eventsTotal.WithLabelValues("eventType", "success").Inc()
			log.Infof("%d records sent to fluent forwarder", 1)
		}
	}
	return nil
}

func parseEventType(r *http.Request) string {
	var eventType = "unknown"
	commonAttributes := firehoseCommonAttributes{}
	if err := json.Unmarshal([]byte(r.Header.Get(commonAttributesHeaderName)), &commonAttributes); err != nil {
		log.Errorf("failed to parse common attributes: %s", err)
	}

	if commonAttributes.CommonAttributes != nil {
		for k, v := range commonAttributes.CommonAttributes {
			log.Debugf("common attribute: %s=%s", k, v)
			if k == eventTypeHeaderName {
				eventType = v
				log.Debugf("set event type to: %s", v)
				break
			}
		}
		log.Debugf("event type is: %s", eventType)
	}
	return eventType
}

func parseRequestBody(r *http.Request) (*firehoseRequestBody, error) {
	body := firehoseRequestBody{}
	logBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("failed to read request body: %s", err)
	}
	r.Body = ioutil.NopCloser(bytes.NewBuffer(logBody))
	if r.Body == nil {
		log.Errorf("request body is empty")
		return nil, errBadReq
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Errorf("failed to decode request body: %s", err)
		return nil, errBadReq
	}
	return &body, nil
}

func JSONHandleError(w http.ResponseWriter, err error) {
	log.Debugf("Firehose error response: %s", err)
	jsonError := func(err APIError) *firehoseResponseBody {
		_, msg, requestID := err.APIError()
		return &firehoseResponseBody{
			ErrorMessage: msg,
			Timestamp:    time.Now().UnixNano() / int64(time.Millisecond),
			RequestID:    requestID,
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if err, ok := err.(APIError); ok {
		code, _, _ := err.APIError()
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(jsonError(err))
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(jsonError(&firehoseAPIError{msg: "internal server error"}))
	}
}
