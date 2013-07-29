package gadget

const (
	// The following constants are all from Firmata.h
	//
	// Message command bytes (128-255 / 0x80-0xFF)
	digitalMessage byte = 0x90 // Send data for a digital pin.
	analogMessage  byte = 0xE0 // Send data for an analog pin (or PWM).
	reportDigital  byte = 0xD0 // Enable digital input by port pair.
	reportAnalog   byte = 0xC0 // Enable analog input by pin #.
	setPinMode     byte = 0xF4 // Set the pin mode.
	reportVersion  byte = 0xF9 // Report protocol version.
	unknown        byte = 0xFF // TODO: Change to system reset as per firmata.h
	startSysex     byte = 0xF0 // Start a MIDI Sysex message
	endSysex       byte = 0xF7 // End a MIDI Sysex message.

	// Extended command set using systex. (0-127 / 0x00-0x7F)
	// 0x00-0x0F reserved for user-defined commands.
	servoConfig           byte = 0x70 // Set max angle, minPulse, maxPulse, freq.
	stringData            byte = 0x71 // A string message with 14-bits per char.
	shiftData             byte = 0x75 // A bitstream to/from a shift register.
	i2cRequest            byte = 0x76 // Send an I2C read/write request.
	i2cReply              byte = 0x77 // A reply to an I2C read request.
	i2cConfig             byte = 0x78 // Config I2C read request.
	extendedAnalog        byte = 0x6F // Analog write (PWM, Servo, etc) to any pin.
	pinStateQuery         byte = 0x6D // Ask for a pin's current mode and value.
	pinStateResponse      byte = 0x6E // Reply with pin's current move and value.
	capabilityQuery       byte = 0x6B // Ask for supported modes and resolution of all pins.
	capabilityResponse    byte = 0x6C // Reply with supported modes and resolution.
	analogMappingQuery    byte = 0x69 // Ask for mapping of analog to pin numbers.
	analogMappingResponse byte = 0x6A // Reply with mapping info.
	reportFirmware        byte = 0x79 // Report name and version of the firmware.
	samplingInterval      byte = 0x7A // Set the poll rate of the main loop.
	sysexNonRealtime      byte = 0x7E // MIDI reserved for non-realtime messages.
	sysexRealtime         byte = 0x7F // MIDI reserved for realtime messages.

	// Pin modes
	INPUT  byte = 0x00 // Digital pin in input mode.
	OUTPUT byte = 0x01 // Digital pin in output mode.
	ANALOG byte = 0x02 // Analog pin in analogInput mode.
	PWM    byte = 0x03 // Digital pin in PWM output mode.
	SERVO  byte = 0x04 // Digital pin in Servo output mode.
	SHIFT  byte = 0x05 // shiftIn/shiftOut mode.
	I2C    byte = 0x06 // Pin included in I2C setup.

	// Pin states
	LOW  state = 0
	HIGH state = 1
)

type state byte