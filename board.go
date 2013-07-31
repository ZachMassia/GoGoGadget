package gadget

import (
	"bufio"
	"bytes"
	//	"encoding/hex"
	"fmt"
	"github.com/ZachMassia/goserial"
	"io"
	"log"
)

type Board struct {
	// Serial connection related.
	cfg    *serial.Config
	fd     uintptr
	buf    *bufio.Reader
	serial io.ReadWriteCloser

	// Firmware information.
	maj, min byte
	firmware string

	msgHandlers cbMap
}

func New(device string) (b *Board, err error) {
	b = &Board{
		cfg: &serial.Config{
			Name: device,
			Baud: defaultBaud,
		},
	}

	b.serial, err, b.fd = serial.OpenPort(b.cfg)
	if err != nil {
		return nil, err
	}

	err = serial.Flush(b.fd, serial.TCIOFLUSH)
	if err != nil {
		b.serial.Close()
		return nil, fmt.Errorf("Error flushing port: %s", err)
	}

	b.buf = bufio.NewReader(b.serial)
	b.init()
	return
}

// Prepares Board b for use. Assumes that the serial connection
// has been properly established.
func (b *Board) init() {
	// Register the callbacks.
	b.msgHandlers = cbMap{
		reportVersion:  b.handleReportVersion,
		reportFirmware: b.handleReportFirmware,
	}
	// Start the message loop.
	go b.run()
}

func (b *Board) run() {
	for {
		msg := message{}
		header, _ := b.buf.ReadByte()

		// Sysex commands have their own header so check for that first.
		switch {
		case header == startSysex:
			// Read until sysexEnd
			data, err := b.buf.ReadBytes(endSysex)
			if err != nil {
				log.Printf("Error reading sysex data: %s", err)
				continue
			}
			msg.t = sysexMsg
			msg.data = append([]byte{header}, data...)

		case bytes.Contains(midiHeaders, []byte{header}):
			// Read MIDI lsb and msb
			lsb, err := b.buf.ReadByte()
			if err != nil {
				log.Printf("Error reading MIDI lsb: %s", err)
				continue
			}
			msb, err := b.buf.ReadByte()
			if err != nil {
				log.Printf("Error reading MIDI msb: %s", err)
				continue
			}
			msg.t = midiMsg
			msg.data = []byte{header, lsb, msb}
		}
		b.handleCallback(msg)
	}
}

func (b *Board) handleCallback(msg message) {
	var cmd byte

	switch msg.t {
	case midiMsg:
		// Check for multibyte MIDI message.
		if msg.data[0] < 0xF0 {
			cmd = msg.data[0] & 0xF0 // multibyte
		} else {
			cmd = msg.data[0]
		}
	case sysexMsg:
		// The second byte is used as the cmd, since the first contains the sysexStart byte.
		cmd = msg.data[1]
	}
	// Try to call the handler
	if cb, ok := b.msgHandlers[cmd]; ok {
		cb(msg)
	}
}

func (b *Board) String() string {
	return fmt.Sprintf("Arduino on device '%s'", b.cfg.Name)
}

// Close properly closes the serial connection to Board b.
func (b *Board) Close() {
	serial.Flush(b.fd, serial.TCIOFLUSH)
	b.serial.Close()
}

// Version returns the Firmata protocol version.
func (b *Board) Version() string {
	return fmt.Sprintf("%d.%d", b.maj, b.min)
}

// Firmware returns the Firmata firmware information.
func (b *Board) Firmware() string {
	return fmt.Sprintf("%s %s", b.firmware, b.Version())
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
	b.maj = m.data[1]
	b.min = m.data[2]
	//log.Print(b.Version())
}

// Store the response from reportFirmware
func (b *Board) handleReportFirmware(m message) {
	b.firmware = string(m.data[4 : len(m.data)-1])
	//log.Print(b.Firmware())
	//log.Printf("Firmware: -- HEXDUMP -- \n %s", hex.Dump(m.data))
}
