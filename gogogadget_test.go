package gadget

import (
	"flag"
	"strings"
	"testing"
)

var port = flag.String("port", "/dev/ttyACM0", "the port the Arduino is on")

func TestBadDevice(t *testing.T) {
	badPort := "/bad/port"
	_, err := New(badPort)
	if err == nil {
		t.Fatalf("Should not have connected to port '%s'.", badPort)
	}
}

func TestStringer(t *testing.T) {
	b := GetBoardAndErrorCheck(t)
	defer b.Close()
	if !strings.Contains(b.String(), *port) {
		t.Fatalf("Board.String() does not contain '%s'. Got '%s'", *port, b)
	}
}

func GetBoardAndErrorCheck(t *testing.T) *Board {
	b, err := New(*port)
	if err != nil {
		t.Fatalf("Could not connect to board: %s", err)
	}
	return b
}
