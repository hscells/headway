package headway_test

import (
	"github.com/hscells/headway"
	"testing"
)

func TestClient(t *testing.T) {
	client := headway.NewClient("http://localhost:7777", "test")
	err := client.Send(10, 100, "example message")
	if err != nil {
		t.Fatal(err)
	}
}
