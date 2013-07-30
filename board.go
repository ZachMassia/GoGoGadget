package gadget

import (
	"bytes"
	"fmt"
	"github.com/tarm/goserial"
	"io"
	"log"
)

type Board struct {
	ver         Version
	firm        Firmware
	cfg         *serial.Config
	serial      io.ReadWriteCloser
	msgHandlers cbMap
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
	b.init()
	return
}

// Prepares Board b for use. Assumes that the serial connection
// has been properly established.
func (b *Board) init() {
	// Register the callbacks
	b.msgHandlers = cbMap{
		reportVersion:  b.handleReportVersion,
		reportFirmware: b.handleReportFirmware,
	}

	// Start the message handling system
	byteChan := b.read()
	msgChan := b.parse(byteChan)
	go b.handleCallback(msgChan)
}

func (b *Board) String() string {
	return fmt.Sprintf("Arduino on device '%s'", b.cfg.Name)
}

// Close properly closes the serial connection to Board b.
func (b *Board) Close() {
	b.serial.Close()
}

// Version returns the Firmata protocol version.
func (b *Board) Version() Version {
	return b.ver
}

// Firmware returns the Firmata firmware information.
func (b *Board) Firmware() Firmware {
	return b.firm
}

// Returns a read only channel on which the raw incoming bytes are sent.
func (b *Board) read() (out chan byte) {
	out = make(chan byte)
	go func() {
		for {
			buf := make([]byte, 1)
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
func (b *Board) parse(byteChan <-chan byte) (out chan message) {
	out = make(chan message)
	go func() {
		for {
			msg := message{}
			buf := make([]byte, 255)
			buf[0] = <-byteChan // Get the first byte of a message.

			// Sysex commands have their own header so check for that first.
			if buf[0] == startSysex {
				msg.t = sysexMsg

				// Read into buf until sysexEnd
				var i = 1
				for data := range byteChan {
					buf[i] = data
					i++
					if data == endSysex {
						break
					}
				}
				msg.data = buf[:i]
			} else {
				msg.t = midiMsg

				// Make sure the first byte is a valid MIDI header
				for !bytes.Contains(midiHeaders, buf[:1]) {
					buf[0] = <-byteChan
				}
				// Get the rest of the MIDI message
				for i := byte(1); i < lenMidiMsg; i++ {
					buf[i] = <-byteChan
				}
				msg.data = buf[:lenMidiMsg]
			}
			// Send out the parsed message
			out <- msg
		}
	}()
	return
}

func (b *Board) handleCallback(m <-chan message) {
	var key byte
	for msg := range m {
		switch msg.t {
		case midiMsg:
			// Check for multibyte MIDI message.
			if msg.data[0] < 0xF0 {
				key = msg.data[0] & 0xF0 // multibyte
			} else {
				key = msg.data[0]
			}
		case sysexMsg:
			// The second byte is used as the key, since the first contains the sysexStart byte.
			key = msg.data[1]
		}

		// Try to call the handler
		if cb, ok := b.msgHandlers[key]; ok {
			cb(msg)
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

// Store the response from reportVersion
func (b *Board) handleReportVersion(m message) {
	b.ver.Major = m.data[1]
	b.ver.Minor = m.data[2]
}

// Store the response from reportFirmware
func (b *Board) handleReportFirmware(m message) {
	b.firm.V.Major = m.data[2]
	b.firm.V.Minor = m.data[3]
	b.firm.Name = string(m.data[4 : len(m.data)-1])
}
