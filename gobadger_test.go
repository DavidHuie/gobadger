package gobadger

import (
	"testing"
)

func TestEverything(t *testing.T) {
	conn := NewConn("TEST_TOKEN")
	conn.Url = "http://localhost:3000"
	err := conn.Error("test")
	if err != nil {
		t.Error(err)
	}
}
