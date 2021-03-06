package gobadger

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	apiUrl            = "https://api.honeybadger.io/v1/notices"
	apiKeyHeader      = "X-API-Key"
	contentTypeHeader = "application/json"
	acceptHeader      = "application/json"
)

var (
	httpClient    *http.Client
	notifier      *Notifier
	serverDetails *Server

	MalformedRequest  = errors.New("Malformed request")
	LineTraceError    = errors.New("Couldn't get trace")
	JsonEncodingError = errors.New("Json encoding error")
	HttpResponseError = errors.New("Error reading response")
	HttpRequestError  = errors.New("Error making HTTP request")
)

type Conn struct {
	Key    string
	Url    string
	Offset int
}

type Payload struct {
	Notifier *Notifier `json:"notifier"`
	Error    *Error    `json:"error"`
	Server   *Server   `json:"server"`
}

type Server struct {
	ProjectRoot     *ProjectRoot `json:"project_root"`
	EnvironmentName string       `json:"environment_name"`
	Hostname        string       `json:"hostname"`
}

type ProjectRoot struct {
	Path string `json:"path"`
}

type Notifier struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Version string `json:"version"`
}

type Error struct {
	Backtrace []*Backtrace `json:"backtrace"`
	Category  string       `json:"class"`
	Message   string       `json:"message"`
}

type Backtrace struct {
	Number string `json:"number"`
	File   string `json:"file"`
}

// Returns a new "connection" to HoneyBadger
func NewConn(apiKey string, lineTraceOffset int) *Conn {
	return &Conn{
		Key:    apiKey,
		Url:    apiUrl,
		Offset: lineTraceOffset,
	}
}

// Returns the file name and line number for the caller
func (c *Conn) getMetadata() (string, int, error) {
	_, file, line, ok := runtime.Caller(c.Offset + 2)
	if !ok {
		return "", 0, LineTraceError
	}
	return file, line, nil
}

// Logs an error and associated metadata to HoneyBadger
func (c *Conn) Error(category string, message interface{}) error {
	file, line, err := c.getMetadata()

	// We've got bigger problems if we can't get stack
	// information.
	if err != nil {
		return err
	}

	backtrace := &Backtrace{File: file, Number: strconv.Itoa(line)}
	error := &Error{
		Backtrace: []*Backtrace{backtrace},
		Category:  category,
		Message:   fmt.Sprintf("%s", message),
	}
	payload := &Payload{Notifier: notifier, Error: error, Server: serverDetails}
	json_payload, err := json.Marshal(payload)

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

	response, err := httpClient.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusCreated {
		return HttpRequestError
	}

	return nil
}

// Logs similarly to Error, but with a format string
func (c *Conn) Errorf(category string, format_string string, params ...interface{}) error {
	str := fmt.Sprintf(format_string, params...)
	return c.Error(category, str)
}

func init() {
	current_directory, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	// Try to locate the environment in a few different places
	env := ""
	env = os.Getenv("GOENV")
	// Blame NationBuilder for this...
	if env == "" {
		env = os.Getenv("RAILS_ENV")
	}
	if env == "" {
		env = os.Getenv("go")
	}

	project_root := &ProjectRoot{Path: current_directory}
	serverDetails = &Server{EnvironmentName: env, Hostname: hostname, ProjectRoot: project_root}
	notifier = &Notifier{Name: "gobadger", URL: "https://github.com/DavidHuie/gobadger", Version: "0.1"}

	httpClient = &http.Client{
		Timeout: 30 * time.Second,
	}
}
