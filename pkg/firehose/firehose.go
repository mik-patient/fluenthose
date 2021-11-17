package firehose

import (
	"bytes"
	"context"
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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	muxlogrus "github.com/pytimer/mux-logrus"

	fluentclient "github.com/IBM/fluent-forward-go/fluent/client"
	log "github.com/sirupsen/logrus"
)

const (
	accessKeyHeaderName = "X-Firehose-Access-Key"
	signatureHeaderName = "X-Firehose-Signature"
	requestIDHeaderName = "X-Amz-Firehose-Request-Id"
)

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

var (
	errAuth       = &firehoseAPIError{code: http.StatusUnauthorized, msg: "unauthorized"}
	errBadReq     = &firehoseAPIError{code: http.StatusBadRequest, msg: "bad request"}
	forwardClient *fluentclient.Client
	accessKey     string
)

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
// https://docs.aws.amazon.com/ja_jp/firehose/latest/dev/httpdeliveryrequestresponse.html#responseformat
type firehoseResponseBody struct {
	RequestID    string `json:"requestId,omitempty"`
	Timestamp    int64  `json:"timestamp,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

func RunFirehoseServer(address, key, forwardAddress string) {
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

	log.Infof("Fluenthose server listening on %s", address)
	log.Infof("log-level: %s, fowarding to: %s", log.GetLevel(), forwardAddress)

	health := healthcheck.NewHandler()

	health.AddLivenessCheck(
		"forwarder",
		healthcheck.TCPDialCheck(forwardAddress, 50*time.Millisecond))
	health.AddReadinessCheck(
		"forwarder",
		healthcheck.TCPDialCheck(forwardAddress, 50*time.Millisecond))
	logOptions := muxlogrus.LogOptions{
		Formatter: &log.JSONFormatter{},
	}
	loggingMiddleware := muxlogrus.NewLogger(logOptions)

	router := mux.NewRouter()
	router.Use(loggingMiddleware.Middleware)
	router.HandleFunc("/", firehoseHandler)
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
	log.Info("Server Started")

	<-done
	log.Info("Server Stopped")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		forwardClient.Disconnect()
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	log.Print("Server Exited Properly")
}

func firehoseHandler(w http.ResponseWriter, r *http.Request) {
	log.Infof("firehose request received from %s", r.RemoteAddr)
	log.Debugf("request headers: %+v", r.Header)

	if r.Method != http.MethodPost {
		JSONHandleError(w, errBadReq)
		return
	}
	key := r.Header.Get(accessKeyHeaderName)
	if key == "" || key != accessKey {
		JSONHandleError(w, errAuth)
		return

	}
	requestID := r.Header.Get(requestIDHeaderName)
	if requestID == "" {
		JSONHandleError(w, errBadReq)
		return
	}
	resp := firehoseResponseBody{
		RequestID: requestID,
	}
	w.Header().Set("Content-Type", "application/json")
	firehoseReq, err := parseRequestBody(r)
	if err != nil {
		log.Errorf("failed to parse request body: %s", err)
		JSONHandleError(w, errBadReq)
		return
	}
	resp.Timestamp = time.Now().UnixNano() / int64(time.Millisecond)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
	for _, record := range firehoseReq.Records {
		log.Debugf("firehose record: %s", string(record.Data))
		msg := &protocol.Message{
			Tag:       "fluenthose",
			Timestamp: time.Now().UTC().UnixNano() / int64(time.Millisecond),
			Record: map[string]interface{}{
				"data": string(record.Data),
			},
			Options: &protocol.MessageOptions{},
		}
		err := forwardClient.SendMessage(msg)
		if err != nil {
			log.Errorf("failed to send message: %s", err)
		}
	}
}

func parseRequestBody(r *http.Request) (*firehoseRequestBody, error) {
	body := firehoseRequestBody{}
	logBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Errorf("failed to read request body: %s", err)
	}
	log.Debugf("request body: %s", string(logBody))
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
	log.Infof("Firehose error response: %s", err)
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
