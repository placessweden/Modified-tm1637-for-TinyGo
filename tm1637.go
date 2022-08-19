// Package tm1637 provides a driver for the TM1637 4-digit 7-segment LED display.
//
// Datasheet: https://www.mcielectronics.cl/website_MCI/static/documents/Datasheet_TM1637.pdf
//
// 20220819/places.sweden@gmail.com:
// - Changed bit banging pin level strategy to High() and Low() to make it work
//   for all kind of bords, not just Arduino AVR.
// - Changed bit masking tecnique in writeByte()
package tm1637

import (
	"machine"
	"time"
)

// Device wraps the pins of the TM1637.
type Device struct {
	clk        machine.Pin
	dio        machine.Pin
	brightness uint8
}

// New creates a new TM1637 device.
func New(clk machine.Pin, dio machine.Pin, brightness uint8) Device {
	return Device{clk: clk, dio: dio, brightness: brightness}
}

// Configure sets up the pins (for output only)
func (d *Device) Configure() {
	d.clk.Configure(machine.PinConfig{Mode: machine.PinOutput})
	d.dio.Configure(machine.PinConfig{Mode: machine.PinOutput})
	d.clk.Low()
	d.dio.Low()
}

// Brightness sets the brightness of the display (0-7).
func (d *Device) Brightness(brightness uint8) {
	if brightness > 7 {
		brightness = 7
	}
	d.brightness = brightness
	d.writeCmd()
	d.writeDsp()
}

// ClearDisplay clears the display.
func (d *Device) ClearDisplay() {
	d.writeData([]byte{0, 0, 0, 0}, 0)
}

// DisplayText shows a text on the display.
//
// Only the first 4 letters in the array text would be shown.
func (d *Device) DisplayText(text []byte) {
	var sequences []byte
	for i, t := range text {
		if i > 3 {
			break
		}
		sequences = append(sequences, encodeChr(t))
	}
	d.writeData(sequences, 0)
}

// DisplayChr shows a single character (A-Z, a-z)
// on the display at position 0-3.
func (d *Device) DisplayChr(chr byte, pos uint8) {
	if pos > 3 {
		pos = 3
	}
	d.writeData([]byte{encodeChr(chr)}, pos)
}

// DisplayNumber shows a number on the display.
//
// Only 4 rightmost digits of the number would be shown.
//
// For negative numbers, only -999 to -1 would be
// shown with a negaive sign.
func (d *Device) DisplayNumber(num int16) {
	var sequences []byte
	var start int16
	if num < 0 {
		sequences = append(sequences, segments[37])
		num *= -1
		start = 100
		num %= 1000
	} else {
		start = 1000
		num %= 10000
	}
	for i := start; i >= 1; i /= 10 {
		if num >= i {
			n := (num / int16(i)) % 10
			sequences = append(sequences, segments[n])
		} else {
			if i == 1 && num == 0 {
				sequences = append(sequences, segments[0])
			} else {
				sequences = append(sequences, 0)
			}
		}
	}
	d.writeData(sequences, 0)
}

// DisplayDigit shows a single-digit number (0-9)
// at position 0-3.
func (d *Device) DisplayDigit(digit uint8, pos uint8) {
	digit %= 10
	d.writeData([]byte{segments[digit]}, pos)
}

// DisplayClock allows you to display hour and minute numbers
// together with the colon on/off.
func (d *Device) DisplayClock(num1 uint8, num2 uint8, colon bool) {
	var sequences []byte
	num := []uint8{num1 % 100, num2 % 100}
	for k := 0; k < 2; k++ {
		for i := 10; i >= 1; i /= 10 {
			n := (num[k] / uint8(i)) % 10
			sequences = append(sequences, segments[n])
		}
	}
	if colon {
		sequences[1] |= 1 << 7
	}
	d.writeData(sequences, 0)
}

func encodeChr(c byte) byte {
	r := rune(c)
	switch {
	case r == 32:
		return segments[36] // space
	case r == 42:
		return segments[38] // star/degrees
	case r == 45:
		return segments[37] // dash
	case r >= 65 && r <= 90:
		return segments[r-55] // uppercase A-Z
	case r >= 97 && r <= 122:
		return segments[r-87] // lowercase a-z
	case r >= 48 && r <= 57:
		return segments[r-48] // 0-9
	default:
		return byte(0)
	}
}

func delaytm() {
	time.Sleep(time.Microsecond * time.Duration(TM1637_DELAY))
}


func (d *Device) start() {
	d.dio.Low()
	delaytm()
	d.clk.Low()
	delaytm()
}

func (d *Device) stop() {
	d.dio.Low()
	delaytm()
	d.clk.High()
	delaytm()
	d.dio.High()
}

func (d *Device) writeByte(data uint8) {
	for i := 0; i < 8; i++ {
		if ((data >> i) & 1) == 1 {     // New way of masking bits
			d.dio.High()
		} else {
			d.dio.Low()
		}
		delaytm()
		d.clk.High()
		delaytm()
		d.clk.Low()
		delaytm()
	} // end for
	d.clk.Low()
	delaytm()
	d.clk.High()
	delaytm()
	d.clk.Low()
	delaytm()
}

func (d *Device) writeCmd() {
	d.start()
	d.writeByte(TM1637_CMD1)
	d.stop()
}

func (d *Device) writeDsp() {
	d.start()
	d.writeByte(TM1637_CMD3 | TM1637_DSP_ON | d.brightness)
	d.stop()
}

func (d *Device) writeData(segments []byte, position uint8) {
	d.writeCmd()
	d.start()
	d.writeByte(TM1637_CMD2 | position)
	for _, seg := range segments {
		d.writeByte(seg)
	}
	d.stop()
	d.writeDsp()
}

// EOF
