package bme680

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"periph.io/x/conn/v3"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/conn/v3/spi"
)

const (
	AddrVariant byte = 0xF0 // read-only, only defined on BME688
	AddrChipID  byte = 0xD0 // read-only, should contain 0x61

	// calibration ranges

	AddrCal1Start byte = 0xE1
	AddrCal1End   byte = 0xEE
	AddrCal2Start byte = 0x8A
	AddrCal2End   byte = 0xA0

	// control registers from this point on

	AddrReset byte = 0xE0

	AddrConfig   byte = 0x75
	AddrCtrlMeas byte = 0x74
	AddrCtrlHum  byte = 0x72
	AddrCtrlGas1 byte = 0x71
	AddrCtrlGas0 byte = 0x70

	AddrGasWait9 byte = 0x6D
	AddrGasWait8 byte = 0x6C
	AddrGasWait7 byte = 0x6B
	AddrGasWait6 byte = 0x6A
	AddrGasWait5 byte = 0x69
	AddrGasWait4 byte = 0x68
	AddrGasWait3 byte = 0x67
	AddrGasWait2 byte = 0x66
	AddrGasWait1 byte = 0x65
	AddrGasWait0 byte = 0x64

	AddrResHeat9 byte = 0x63
	AddrResHeat8 byte = 0x62
	AddrResHeat7 byte = 0x61
	AddrResHeat6 byte = 0x60
	AddrResHeat5 byte = 0x5F
	AddrResHeat4 byte = 0x5E
	AddrResHeat3 byte = 0x5D
	AddrResHeat2 byte = 0x5C
	AddrResHeat1 byte = 0x5B
	AddrResHeat0 byte = 0x5A

	AddrIdacHeat9 byte = 0x59
	AddrIdacHeat8 byte = 0x58
	AddrIdacHeat7 byte = 0x57
	AddrIdacHeat6 byte = 0x56
	AddrIdacHeat5 byte = 0x55
	AddrIdacHeat4 byte = 0x54
	AddrIdacHeat3 byte = 0x53
	AddrIdacHeat2 byte = 0x52
	AddrIdacHeat1 byte = 0x51
	AddrIdacHeat0 byte = 0x50

	// data registers from this point on unless marked otherwise

	AddrGasRLSB    byte = 0x2B // mixed status/data
	AddrGasRMSB    byte = 0x2A
	AddrHumLSB     byte = 0x26
	AddrHumMSB     byte = 0x25
	AddrTempXLSB   byte = 0x24
	AddrTempLSB    byte = 0x23
	AddrTempMSB    byte = 0x22
	AddrPressXLSB  byte = 0x21
	AddrPressLSB   byte = 0x20
	AddrPressMSB   byte = 0x1F
	AddrEasStatus0 byte = 0x1D // status only
)

// Oversampling affects how much time is taken to measure each of temperature,
// pressure and humidity.
//
// Using high oversampling and low standby results in highest power
// consumption, but this is still below 1mA so we generally don't care.
type Oversampling uint8

// Possible oversampling values.
//
// The higher the more time and power it takes to take a measurement. Even at
// 16x for all 3 sensors, it is less than 100ms albeit increased power
// consumption may increase the temperature reading.
const (
	Off  Oversampling = 0
	O1x  Oversampling = 1
	O2x  Oversampling = 2
	O4x  Oversampling = 3
	O8x  Oversampling = 4
	O16x Oversampling = 5
)

const oversamplingName = "Off1x2x4x8x16x"

var oversamplingIndex = [...]uint8{0, 3, 5, 7, 9, 11, 14}

func (o Oversampling) String() string {
	if o >= Oversampling(len(oversamplingIndex)-1) {
		return fmt.Sprintf("Oversampling(%d)", o)
	}
	return oversamplingName[oversamplingIndex[o]:oversamplingIndex[o+1]]
}

func (o Oversampling) asValue() int {
	switch o {
	case O1x:
		return 1
	case O2x:
		return 2
	case O4x:
		return 4
	case O8x:
		return 8
	case O16x:
		return 16
	default:
		return 0
	}
}

// Filter specifies the internal IIR filter to get steadier measurements.
//
// Oversampling will get better measurements than filtering but at a larger
// power consumption cost, which may slightly affect temperature measurement.
type Filter uint8

// Possible filtering values.
//
// The higher the filter, the slower the value converges but the more stable
// the measurement is.
const (
	NoFilter Filter = 0
	F2       Filter = 1
	F4       Filter = 2
	F8       Filter = 3
	F16      Filter = 4
	F32      Filter = 5
	F64      Filter = 6
	F128     Filter = 7
)

// DefaultOpts is the recommended default options.
var DefaultOpts = Opts{
	Temperature: O4x,
	Pressure:    O4x,
	Humidity:    O4x,
}

// Opts defines the options for the device.
//
// Recommended sensing settings as per the datasheet:
//
// → Weather monitoring: manual sampling once per minute, all sensors O1x.
// Power consumption: 0.16µA, filter NoFilter. RMS noise: 3.3Pa / 30cm, 0.07%RH.
//
// → Humidity sensing: manual sampling once per second, pressure Off, humidity
// and temperature O1X, filter NoFilter. Power consumption: 2.9µA, 0.07%RH.
//
// → Indoor navigation: continuous sampling at 40ms with filter F16, pressure
// O16x, temperature O2x, humidity O1x, filter F16. Power consumption 633µA.
// RMS noise: 0.2Pa / 1.7cm.
//
// → Gaming: continuous sampling at 40ms with filter F16, pressure O4x,
// temperature O1x, humidity Off, filter F16. Power consumption 581µA. RMS
// noise: 0.3Pa / 2.5cm.
//
// See the datasheet for more details about the trade offs.
type Opts struct {
	// Temperature must be measured for pressure and humidity to be measured.
	Temperature Oversampling
	// Pressure can be oversampled up to 8x on BMP180 and 16x on BME280/BMP280.
	Pressure Oversampling
	Humidity Oversampling
	// Filter is only used while using SenseContinuous()
	Filter Filter
}

// NewI2C returns an object that communicates over I²C to BMP180/BME280/BMP280
// environmental sensor.
//
// The address must be 0x76 or 0x77. BMP180 uses 0x77. BME280/BMP280 default to
// 0x76 and can optionally use 0x77. The value used depends on HW
// configuration of the sensor's SDO pin.
//
// It is recommended to call Halt() when done with the device so it stops
// sampling.
func NewI2C(b i2c.Bus, addr uint16, opts *Opts) (*Dev, error) {
	switch addr {
	case 0x76, 0x77:
	default:
		return nil, errors.New("bme680: given address not supported by device")
	}
	d := &Dev{d: &i2c.Dev{Bus: b, Addr: addr}, isSPI: false}
	if err := d.makeDev(opts); err != nil {
		return nil, err
	}
	return d, nil
}

// NewSPI returns an object that communicates over SPI to either a BME280 or
// BMP280 environmental sensor.
//
// It is recommended to call Halt() when done with the device so it stops
// sampling.
//
// When using SPI, the CS line must be used.
func NewSPI(p spi.Port, opts *Opts) (*Dev, error) {
	// It works both in Mode0 and Mode3.
	c, err := p.Connect(10*physic.MegaHertz, spi.Mode3, 8)
	if err != nil {
		return nil, fmt.Errorf("bme680: %v", err)
	}
	d := &Dev{d: c, isSPI: true}
	if err := d.makeDev(opts); err != nil {
		return nil, err
	}
	return d, nil
}

// Dev is a handle to an initialized BME680 device.
//
// The actual device type was auto detected.
type Dev struct {
	d           conn.Conn
	isSPI       bool
	is688       bool
	opts        Opts
	name        string
	calibration calibration680

	mu   sync.Mutex
	stop chan struct{}
	wg   sync.WaitGroup
}

func (d *Dev) String() string {
	return fmt.Sprintf("%s{%s}", d.name, d.d)
}

// Sense requests a one time measurement as °C, kPa and % of relative humidity.
//
// The very first measurements may be of poor quality.
func (d *Dev) Sense(e *physic.Env) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.stop != nil {
		return d.wrap(errors.New("already sensing continuously"))
	}

	err := d.writeCommands([]byte{
		// ctrl_meas
		AddrCtrlMeas, byte(d.opts.Temperature)<<5 | byte(d.opts.Pressure)<<2 | byte(forced),
	})
	if err != nil {
		return d.wrap(err)
	}

	return d.sense680(e)
}

// SenseContinuous returns measurements as °C, kPa and % of relative humidity
// on a continuous basis.
//
// The application must call Halt() to stop the sensing when done to stop the
// sensor and close the channel.
//
// It's the responsibility of the caller to retrieve the values from the
// channel as fast as possible, otherwise the interval may not be respected.
func (d *Dev) SenseContinuous(interval time.Duration) (<-chan physic.Env, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.stop != nil {
		// Don't send the stop command to the device.
		close(d.stop)
		d.stop = nil
		d.wg.Wait()
	}

	err := d.writeCommands([]byte{
		AddrConfig, byte(d.opts.Filter) << 2,
		AddrCtrlMeas, byte(d.opts.Temperature)<<5 | byte(d.opts.Pressure)<<2 | byte(forced),
	})
	if err != nil {
		return nil, d.wrap(err)
	}

	sensing := make(chan physic.Env)
	d.stop = make(chan struct{})
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer close(sensing)
		d.sensingContinuous(interval, sensing, d.stop)
	}()
	return sensing, nil
}

// Precision implements physic.SenseEnv.
func (d *Dev) Precision(e *physic.Env) {}

// Halt stops the BME680 from acquiring measurements as initiated by
// SenseContinuous().
//
// It is recommended to call this function before terminating the process to
// reduce idle power usage and a goroutine leak.
func (d *Dev) Halt() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.stop == nil {
		return nil
	}
	close(d.stop)
	d.stop = nil
	d.wg.Wait()

	return d.writeCommands([]byte{
		// config
		AddrConfig, byte(NoFilter) << 2,
		// ctrl_meas
		AddrCtrlMeas, byte(d.opts.Temperature)<<5 | byte(d.opts.Pressure)<<2 | byte(sleep),
	})
}

//

func (d *Dev) makeDev(opts *Opts) error {
	d.opts = *opts

	var variantID, chipID [1]byte

	// 0xF0 and 0xD0 hold the variant ID (assumed to be 0 for BME680, 1 for BME688) and chip ID (0x61) respectively
	if err := d.readReg(AddrVariant, variantID[:]); err != nil {
		return err
	}
	if err := d.readReg(AddrChipID, chipID[:]); err != nil {
		return err
	}

	if chipID[0] != 0x61 {
		return fmt.Errorf("bme680: unexpected chip id %x", chipID[0])
	}

	if variantID[0] == 1 {
		d.name = "BME688"
		d.is688 = true
	} else {
		d.name = "BME680"
	}

	var cal1 [(AddrCal1End + 1) - AddrCal1Start]byte
	if err := d.readReg(AddrCal1Start, cal1[:]); err != nil {
		return err
	}
	// Read calibration data h2~6
	var cal2 [(AddrCal2End + 1) - AddrCal2Start]byte
	if err := d.readReg(AddrCal2Start, cal2[:]); err != nil {
		return err
	}

	d.calibration = newCalibration(cal1[:], cal2[:])
	b := []byte{
		// ctrl_meas; put it to sleep otherwise the config update may be
		// ignored. This is really just in case the device was somehow put
		// into normal but was not Halt'ed.
		0xF4, byte(d.opts.Temperature)<<5 | byte(d.opts.Pressure)<<2 | byte(sleep),
		// ctrl_hum
		0xF2, byte(d.opts.Humidity),
		// config
		0xF5, byte(NoFilter) << 2,
		// ctrl_meas must be re-written last.
		0xF4, byte(d.opts.Temperature)<<5 | byte(d.opts.Pressure)<<2 | byte(sleep),
	}

	return d.writeCommands(b)
}

func (d *Dev) sensingContinuous(interval time.Duration, sensing chan<- physic.Env, stop <-chan struct{}) {
	t := time.NewTicker(interval)
	defer t.Stop()

	var err error
	for {
		// Do one initial sensing right away.
		e := physic.Env{}
		d.mu.Lock()
		err = d.sense680(&e)
		d.mu.Unlock()
		if err != nil {
			log.Printf("%s: failed to sense: %v", d, err)
			return
		}
		select {
		case sensing <- e:
		case <-stop:
			return
		}
		select {
		case <-stop:
			return
		case <-t.C:
		}
	}
}

func (d *Dev) readReg(reg uint8, b []byte) error {
	if d.isSPI {
		// MSB is 0 for write and 1 for read.
		read := make([]byte, len(b)+1)
		write := make([]byte, len(read))
		// Rest of the write buffer is ignored.
		write[0] = reg
		if err := d.d.Tx(write, read); err != nil {
			return d.wrap(err)
		}
		copy(b, read[1:])
		return nil
	}
	if err := d.d.Tx([]byte{reg}, b); err != nil {
		return d.wrap(err)
	}
	return nil
}

// writeCommands writes a command to the device.
//
// Warning: b may be modified!
func (d *Dev) writeCommands(b []byte) error {
	if d.isSPI {
		// set RW bit 7 to 0.
		for i := 0; i < len(b); i += 2 {
			b[i] &^= 0x80
		}
	}
	if err := d.d.Tx(b, nil); err != nil {
		return d.wrap(err)
	}
	return nil
}

func (d *Dev) wrap(err error) error {
	return fmt.Errorf("%s: %v", strings.ToLower(d.name), err)
}

var doSleep = time.Sleep

var _ conn.Resource = &Dev{}
var _ physic.SenseEnv = &Dev{}
