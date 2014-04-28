package gobadger

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"strings"
)

const (
	apiUrl            = "https://api.honeybadger.io/v1/notices"
	apiKeyHeader      = "X-API-Key"
	contentTypeHeader = "application/json"
	acceptHeader      = "application/json"
)

var (
	lineTraceOffset = 2

	// Use the same notifier JSON for all requests
	notifier *Notifier

	MalformedRequest  = errors.New("Malformed request")
	LineTraceError    = errors.New("Couldn't get trace")
	JsonEncodingError = errors.New("Json encoding error")
	HttpResponseError = errors.New("Error reading response")
	HttpRequestError  = errors.New("Error making HTTP request")
)

type Conn struct {
	Key string
	Url string
}

type Payload struct {
	Notifier *Notifier `json:"notifier"`
	Error    *Error    `json:"error"`
}

type Notifier struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Version string `json:"version"`
}

type Error struct {
	Message   string       `json:"message"`
	Backtrace []*Backtrace `json:"backtrace"`
}

type Backtrace struct {
	Number string `json:"number"`
	File   string `json:"file"`
}

// Returns a new "connection" to HoneyBadger
func NewConn(api_key string) *Conn {
	return &Conn{Key: api_key, Url: apiUrl}
}

// Returns the file name and line number for the caller
func getMetadata() (string, int, error) {
	_, file, line, ok := runtime.Caller(lineTraceOffset)
	if !ok {
		return "", 0, LineTraceError
	}
	return file, line, nil
}

// Logs an error and associated metadata to HoneyBadger
func (c *Conn) Error(message string) error {
	file, line, err := getMetadata()

	// We've got bigger problems if we can't get stack
	// information.
	if err != nil {
		return err
	}

	backtrace := &Backtrace{File: file, Number: strconv.Itoa(line)}
	error := &Error{Message: message, Backtrace: []*Backtrace{backtrace}}
	payload := &Payload{Notifier: notifier, Error: error}
	json_payload, err := json.Marshal(payload)

	log.Print(string(json_payload))

	if err != nil {
		return JsonEncodingError
	}

	request, err := http.NewRequest("POST", c.Url, strings.NewReader(string(json_payload)))
	if err != nil {
		return MalformedRequest
	}

	// Set required headers
	request.Header.Set(apiKeyHeader, c.Key)
	request.Header.Set("Content-Type", contentTypeHeader)
	request.Header.Set("Accept", acceptHeader)

	client := &http.Client{}

	response, err := client.Do(request)
	if (err != nil) || (response.StatusCode != http.StatusCreated) {
		return HttpRequestError
	}

	return nil
}

func init() {
	notifier = &Notifier{Name: "gobadger", URL: "https://github.com/DavidHuie/gobadger", Version: "0.1"}
}
