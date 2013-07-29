package gadget

import (
	"fmt"
	"github.com/tarm/goserial"
	"io"
)

type Board struct {
	cfg    *serial.Config
	serial io.ReadWriteCloser
}

func New(device string) (b *Board, err error) {
	b = &Board{
		cfg: &serial.Config{
			Name: device,
			Baud: defaultBaud,
		},
	}
	b.serial, err = serial.OpenPort(b.cfg)
	if err != nil {
		return nil, err
	}
	return
}

func (b *Board) String() string {
	return fmt.Sprintf("Arduino on device '%s'", b.cfg.Name)
}

// Close properly closes the serial connection to Board b.
func (b *Board) Close() { b.serial.Close() }
