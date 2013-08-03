package gadget

import (
	"bytes"
	"fmt"
	"io"
)

type state byte // Compile time check for high/low.

const (
	// Pin modes
	INPUT  byte = iota // Digital pin in input mode.
	OUTPUT             // Digital pin in output mode.
	ANALOG             // Analog pin in analogInput mode.
	PWM                // Digital pin in PWM output mode.
	SERVO              // Digital pin in Servo output mode.
	SHIFT              // shiftIn/shiftOut mode.
	I2C                // Pin included in I2C setup.

	// Pin states
	LOW  state = 0
	HIGH state = 1
)

var (
	// String representation of pin mode bytes.
	PinModeString = map[byte]string{
		INPUT:  "INPUT",
		OUTPUT: "OUTPUT",
		ANALOG: "ANALOG",
		PWM:    "PWM",
		SERVO:  "SERVO",
		SHIFT:  "SHIFT",
		I2C:    "I2C",
	}

	// Slice of all valid pin modes.
	validPinModes = []byte{INPUT, OUTPUT, ANALOG, PWM, SERVO, SHIFT, I2C}
)

// Reads the slice of mode+res pairs for a single pin.
func unpackPinModeDataSlice(data []byte) (modes []byte) {
	// Must be even number of elements
	if len(data)%2 != 0 {
		return nil
	}

	// Unpack the data
	for i := 0; i < len(data); i += 2 {
		modes = append(modes, data[i])
	}
	return
}

type pin struct {
	// The serial port the board is communicating on.
	serial io.Writer

	// The pins number. For analog pins this is the
	// real number, not the Arduino style A0-A15.
	num byte

	// Pins in analog mode must be refered to based on
	// the Arduino style A0. This number has no use for
	// digital only pins and will be set to 0.
	analogNum byte

	// When a pin is in digital in/out mode, reporting
	// is based on ports containing 8 pins. This is
	// the pins port number.
	port byte

	mode           byte   // The current mode.
	reporting      bool   // Is the pin (or port in digital mode) reporting.
	supportedModes []byte // The valid modes for this pin.
}

// Returns an analog pin.
func newPin(s io.Writer, pinNum, aPinNum byte, modes []byte) (p *pin) {
	p = &pin{
		serial:         s,
		num:            pinNum,
		analogNum:      aPinNum,
		port:           pinToPort(pinNum),
		supportedModes: modes,
	}

	// Set the default pin mode.
	if bytes.Contains(p.supportedModes, []byte{ANALOG}) {
		p.setMode(ANALOG)
		// Analog pins report by default. Turn it off
		// until requested by the user.
		p.setReporting(false)
	} else {
		p.setMode(OUTPUT) // Digital pin default.
	}
	return
}

// Set the mode of pin p.
func (p *pin) setMode(mode byte) (err error) {
	// Error checking
	switch {
	case !bytes.Contains(validPinModes, []byte{mode}):
		return fmt.Errorf("Pin mode %X not valid", mode)

	case !bytes.Contains(p.supportedModes, []byte{mode}):
		return fmt.Errorf("Pin mode %s not supported by pin %d", PinModeString[mode], p.num)

	case mode == p.mode:
		// TODO: Should this return nil? Technically not an error.
		return fmt.Errorf("Pin %d already in %s mode", p.num, PinModeString[mode])
	}

	// Update the pins mode flag.
	p.mode = mode

	// Send the message.
	msg := []byte{setPinMode, p.num, mode}
	p.serial.Write(msg)
	return
}

func (p *pin) setReporting(newState bool) (err error) {
	// Do not turn on reporting for non input pin.
	if newState && (p.mode != INPUT && p.mode != ANALOG) {
		return fmt.Errorf("Pin %d not in INPUT or ANALOG mode", p.num)
	}
	p.reporting = newState

	var msg []byte

	// Create the message to send
	switch p.mode {
	case ANALOG:
		msg = []byte{
			reportAnalog | p.analogNum,
			boolToByte(newState),
		}
	case INPUT:
	case OUTPUT:
		// TODO: This is only a temporary solution.
		//       Proper checking for pins in modes
		//       other than INPUT/OUPUT should be done.
		msg = []byte{
			reportDigital | p.port,
			boolToByte(newState),
		}
	}
	p.serial.Write(msg)
	return
}
