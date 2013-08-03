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

	// A mapping of normal pin number to their analog (A0 style) numbers.
	// A value of 0x7F (127) means the pin is digital only.
	analogMapping map[byte]byte

	// A mapping of message handlers, the key is the command byte.
	msgHandlers cbMap

	// Used to notify when the firmware reponse comes in and the
	// board is ready to communicate.
	boardDoneReboot chan bool

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
		ready:           make(chan bool),
		boardDoneReboot: make(chan bool),
		analogPins:      make(map[byte]*pin),
		digitalPins:     make(map[byte]*pin),
		analogMapping:   make(map[byte]byte),
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

	// Once init returns, we have received all required
	// setup messages from the board.
	b.init()

	return
}

// Prepares Board b for use. Assumes that the serial connection
// has been properly established.
func (b *Board) init() {
	// Register the callbacks.
	b.msgHandlers = cbMap{
		reportVersion:         b.handleReportVersion,
		reportFirmware:        b.handleReportFirmware,
		capabilityResponse:    b.handleCapabilityResponse,
		analogMappingResponse: b.handleAnalogMappingResponse,
		analogMessage:         b.handleAnalogMessage,
		digitalMessage:        b.handleDigitalMessage,
	}
	// Start the message loop.
	go b.run()

	for {
		select {
		case <-b.boardDoneReboot:
			b.sendAnalogMappingQuery()
			b.sendCapabilityQuery()

		case <-b.ready:
			return
		}
	}
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
			b.handleCallback(msg)

		default:
			// Read the two MIDI data bytes
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
			b.handleCallback(msg)
		}
	}
}

func (b *Board) handleCallback(msg message) {
	var cmd byte

	switch msg.t {
	case midiMsg:
		// Some messages include data in the command byte, but
		// here we only need the base command.
		if msg.data[0] < 0xF0 {
			cmd = msg.data[0] & 0xF0 // Remove data from the command
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
		// Nothing to do here.
		return
	}

	// The analogMappingReponse must be handled before the pins
	// can be initialized.
	if len(b.analogMapping) == 0 {
		// Resend the analog mapping request.
		b.sendAnalogMappingQuery()
		// Give it a a seconds then try to init the pins again by
		// sending another capability request.
		//
		// TODO: Store the maps passed to init so that it can be called
		//       directly without sending another request to the board.
		time.AfterFunc(1*time.Second, func() { b.sendCapabilityQuery() })
		return
	}

	// Initialize the analog pins.
	for pin, modes := range analog {
		if analogNum, ok := b.analogMapping[pin]; ok {
			b.analogPins[pin] = newPin(b.serial, pin, analogNum, modes)
		} else {
			log.Printf("Error initializing analog pin %d", pin)
		}
	}

	// Intialize the digital pins.
	for pin, modes := range digital {
		// 0x7F is passed directly as the analog pin number
		// since it does not apply to digital pins.
		b.digitalPins[pin] = newPin(b.serial, pin, 0x7F, modes)
	}

	// Send the ready message to New() so it can return.
	b.ready <- true

	// Ignore any furthur calls from the capabilityResponse handler.
	b.pinsInitialized = true
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
// Takes an analog pin number between 0-15.
func (b *Board) AnalogRead(pin byte) byte {
	// TODO: Implement
	return 0
}

// AnalogWrite sets the PWM out value of the analog pin.
// Takes an analog pin number between 0-15.
func (b *Board) AnalogWrite(pin, val byte) {
	// TODO: Implement
}

func (b *Board) SetPinMode(pin, mode byte) {

}

// -- Message Sending Functions -- //

// Wraps a message in sysex start/end bytes, and writes it
// to the serial port.
func (b *Board) sendSysex(msg []byte) (n int, err error) {
	m := wrapInSysex(msg)
	n, err = b.serial.Write(m)
	return

}

func (b *Board) sendCapabilityQuery()    { b.sendSysex([]byte{capabilityQuery}) }
func (b *Board) sendAnalogMappingQuery() { b.sendSysex([]byte{analogMappingQuery}) }

// -- Message Handling Functions -- //

func (b *Board) handleAnalogMessage(m message) {
	log.Printf("ANALOG PIN %d VAL %d", m.data[0]&0x0F, m.data[0]|m.data[1]<<7)
}

func (b *Board) handleDigitalMessage(m message) {
	port := m.data[0] & 0x0F
	lsb, msb := m.data[1], m.data[2]
	portVal := lsb | msb<<7
	log.Printf("DIGITAL PORT %X VAL %b", port, portVal)
}

// Store the response from reportVersion
func (b *Board) handleReportVersion(m message) {
	b.maj = m.data[1]
	b.min = m.data[2]
}

// Store the response from reportFirmware.
func (b *Board) handleReportFirmware(m message) {
	b.firmware = string(m.data[4 : len(m.data)-1])

	if !b.pinsInitialized {
		// Let the init() func continue setting up the pins.
		b.boardDoneReboot <- true
	}
}

// Parse the capability response and pass to initPins.
func (b *Board) handleCapabilityResponse(m message) {
	// Maps of pin# -> supported modes
	analogPins := make(map[byte][]byte)
	digitalPins := make(map[byte][]byte)

	// Create a buffer containing just the pin mode (bytes 2 to END-1)
	currentPin := byte(0)
	buf := bytes.NewBuffer(m.data[2 : len(m.data)-1])
	for buf.Len() > 0 {
		d, _ := buf.ReadBytes(0x7F)
		info := unpackPinModeDataSlice(d[:len(d)-1]) // drop the 0x7F delimiter

		switch {
		case bytes.Contains(info, []byte{ANALOG}):
			analogPins[currentPin] = info

		case bytes.Contains(info, []byte{INPUT, OUTPUT}):
			digitalPins[currentPin] = info
		}
		currentPin++
	}
	b.initPins(analogPins, digitalPins)
}

// Sets the analogMapping values.
func (b *Board) handleAnalogMappingResponse(m message) {
	// For each key value pair, the key is the regular pin number, and
	// the value is the analog pin number, or 0x7F (127) if the pin
	// does not support analog.
	for pin, num := range m.data[2 : len(m.data)-1] {
		if num != 0x7F {
			b.analogMapping[byte(pin)] = num

			// Hack until I figure out why Firmata sends
			// A13 (pin 67) as A10.
			// TODO: FIXME
			if pin == 67 {
				b.analogMapping[byte(pin)] = 13
			}
		}
	}
}
