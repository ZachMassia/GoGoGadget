package gadget

import (
	"fmt"
	"github.com/tarm/goserial"
	"io"
	"log"
	"time"
)

type Board struct {
	cfg    *serial.Config
	serial io.ReadWriteCloser

	msgHandlers [255]callback
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

	time.AfterFunc(initDelay, b.init)
	return
}

// Prepares Board b for use. Assumes that the serial connection
// has been properly established.
func (b *Board) init() {
	byteChan := b.read()
	msgChan := b.parse(byteChan)
	go b.handleCallback(msgChan)
}

func (b *Board) String() string {
	return fmt.Sprintf("Arduino on device '%s'", b.cfg.Name)
}

// Close properly closes the serial connection to Board b.
func (b *Board) Close() { b.serial.Close() }

// Returns a read only channel on which the raw incoming bytes are sent.
func (b *Board) read() (out chan byte) {
	out = make(chan byte)
	go func() {
		buf := make([]byte, 1)
		for {
			// Read blocks until a byte is returned.
			// Once it is received, send it down the processing chain.
			if _, err := b.serial.Read(buf); err != nil {
				log.Printf("Read err: %s", err)
			}
			out <- buf[0]
		}
	}()
	return
}

// Parse raw bytes into messages and send them out.
func (b *Board) parse(bytes <-chan byte) (out chan message) {
	out = make(chan message)
	go func() {
		for {
			// TODO: parse bytes into message
		}
	}()
	return
}

func (b *Board) handleCallback(m <-chan message) {
	// The callback index.
	var i byte

	for msg := range m {
		switch msg.t {
		case midiMsg:
			// Check for multibyte MIDI message.
			if msg.data[0] < 0xF0 {
				i = msg.data[0] & 0xF0 // multibyte
			} else {
				i = msg.data[0]
			}

		case sysexMsg:
			// The second byte is used as the index, as the first
			// contains the sysexStart byte.
			i = msg.data[1]

		}
		// Try to call the handler
		if b.msgHandlers[i] != nil {
			b.msgHandlers[i](msg)
		}
	}
}

// DigitalRead returns the state of the digital pin.
func (b *Board) DigitalRead(pin byte) state {
	return 0
}

// DigitalWrite sets the state of the digital pin.
func (b *Board) DigitalWrite(pin byte, s state) {
	// TODO: Implement
}

// AnalogRead returns the value of the analog pin.
func (b *Board) AnalogRead(pin byte) byte {
	// TODO: Implement
	return 0
}

// AnalogWrite sets the PWM out value of the analog pin.
func (b *Board) AnalogWrite(pin, val byte) {
	// TODO: Implement
}
