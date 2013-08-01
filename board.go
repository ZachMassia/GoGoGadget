package gadget

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/ZachMassia/goserial"
	"io"
	"log"
	"time"
)

type Board struct {
	cfg    *serial.Config     // Port and baud rate
	fd     uintptr            // Serial port file descriptor.
	buf    *bufio.Reader      // Buffered reading from serial.
	serial io.ReadWriteCloser // The serial connection.

	maj, min byte   // Firmware version
	firmware string // The name of the sketch uploaded to the board.

	// Has the initial pin capability response been handled.
	pinsInitialized bool

	// The pins are stored in structs, with the key being that pins number.
	// Analog pins do not use the A0 numbering.
	analogPins, digitalPins map[byte]*pin

	// A mapping of message handlers, the key is the command byte.
	msgHandlers cbMap

	// This channel is used to tell New that the board received
	// the capability response and the board is fully configured
	// and ready to return.
	ready chan bool
}

// New returns a fully configured Board, with the message handling
// loop running in it's own goroutine.
func New(device string) (b *Board, err error) {
	b = &Board{
		cfg: &serial.Config{
			Name: device,
			Baud: defaultBaud,
		},
		ready:       make(chan bool),
		analogPins:  make(map[byte]*pin),
		digitalPins: make(map[byte]*pin),
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

	// Wait for the board to be configured.
	<-b.ready
	close(b.ready)

	return
}

// Prepares Board b for use. Assumes that the serial connection
// has been properly established.
func (b *Board) init() {
	// Register the callbacks.
	b.msgHandlers = cbMap{
		reportVersion:      b.handleReportVersion,
		reportFirmware:     b.handleReportFirmware,
		capabilityResponse: b.handleCapabilityResponse,
	}
	// Start the message loop.
	go b.run()

	time.AfterFunc(3*time.Second, b.sendCapabilityQuery)
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

// Initializes the pins if it has not already been done.
func (b *Board) initPins(analog, digital map[byte][]byte) {
	if b.pinsInitialized {
		return
	}
	b.pinsInitialized = true

	// Initialize the analog pins.
	for pin, modes := range analog {
		b.analogPins[pin] = newPin(b.serial, pin, modes)
	}

	// Intialize the digital pins.
	for pin, modes := range digital {
		b.digitalPins[pin] = newPin(b.serial, pin, modes)
	}

	// Send the ready message to New() so it can return.
	b.ready <- true
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

// -- Message Sending Functions -- //

func (b *Board) sendCapabilityQuery() {
	msg := wrapInSysex([]byte{capabilityQuery})
	b.serial.Write(msg)
}

// -- Message Handling Functions -- //

// Store the response from reportVersion
func (b *Board) handleReportVersion(m message) {
	b.maj = m.data[1]
	b.min = m.data[2]
}

// Store the response from reportFirmware.
func (b *Board) handleReportFirmware(m message) {
	b.firmware = string(m.data[4 : len(m.data)-1])
}

// Parse the capability response and pass to initPins.
func (b *Board) handleCapabilityResponse(m message) {
	currentPin := byte(0)
	analogPins := make(map[byte][]byte)
	digitalPins := make(map[byte][]byte)

	// Create a buffer containing just the pin mode (bytes 2 to END-1)
	buf := bytes.NewBuffer(m.data[2 : len(m.data)-1])
	for buf.Len() > 0 {
		d, _ := buf.ReadBytes(0x7F)
		info := unpackPinModeDataSlice(d[:len(d)-1]) // drop the 0x7F delimiter

		switch {
		case bytes.Contains(d, []byte{ANALOG}):
			analogPins[currentPin] = info
		case bytes.Contains(d, []byte{DIGITAL}):
			digitalPins[currentPin] = info
		}
		currentPin++
	}
	b.initPins(analogPins, digitalPins)
}
