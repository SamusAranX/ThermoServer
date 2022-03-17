// Copyright 2016 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package bme680

import (
	"periph.io/x/conn/v3/physic"
)

const (
	HectoPascal = 100 * physic.Pascal

	tempPrecision = 1000000
)

// sense680 reads the device's registers for bme680.
//
// It must be called with d.mu lock held.
func (d *Dev) sense680(e *physic.Env) error {
	buf := [8]byte{}
	b := buf[:]
	if err := d.readReg(AddrPressMSB, b); err != nil {
		return err
	}

	// These values are 20 bits as per doc.
	pRaw := uint32(buf[0])<<12 | uint32(buf[1])<<4 | uint32(buf[2])>>4
	tRaw := uint32(buf[3])<<12 | uint32(buf[4])<<4 | uint32(buf[5])>>4
	hRaw := uint32(uint16(uint32(buf[6])<<8 | uint32(buf[7])))

	tempComp, tempFine := d.calibration.compensateTemp(tRaw)

	e.Temperature = physic.Temperature(tempComp*tempPrecision)*physic.Kelvin/tempPrecision + physic.ZeroCelsius

	if d.opts.Pressure != Off {
		pressureComp := d.calibration.compensatePressure(pRaw, tempFine)
		e.Pressure = physic.Pressure(pressureComp) * physic.Pascal
	}

	if d.opts.Humidity != Off {
		// the compensation code is Fucked
		// TODO: get a better humidity sensor and figure out what's wrong
		humidityComp := d.calibration.compensateHumidity(hRaw, tempFine/5120/100000)
		e.Humidity = physic.RelativeHumidity(humidityComp) * physic.PercentRH
		//e.Humidity = physic.RelativeHumidity(humidityComp) * 10000 / 1024 * physic.MicroRH
	}

	return nil
}

func (d *Dev) isIdle680() (bool, error) {
	// status
	v := [1]byte{}
	if err := d.readReg(AddrEasStatus0, v[:]); err != nil {
		return false, err
	}
	// Make sure bit 5 is cleared. Bit 0 is only important at device boot up.
	return v[0]&0b100000 == 0, nil
}

// mode is the operating mode.
type mode byte

const (
	sleep    mode = 0 // no operation, all registers accessible, lowest power, selected after startup
	forced   mode = 1 // perform one measurement, store results and return to sleep mode
	parallel mode = 2 // only supported on BME688
)

// newCalibration parses calibration data from both buffers.
func newCalibration(cd1, cd2 []byte) (c calibration680) {
	// cd1 covers 0xE1 through 0xEE
	// cd2 covers 0x8A through 0xA0

	getInt16 := func(lsb, msb byte) int16 {
		return int16(lsb) | (int16(msb) << 8)
	}

	getUInt16 := func(lsb, msb byte) uint16 {
		return uint16(lsb) | (uint16(msb) << 8)
	}

	c.t1 = getUInt16(cd1[8], cd1[9])
	c.t2 = getInt16(cd2[0], cd2[1])
	c.t3 = int8(cd2[2])

	c.p1 = getUInt16(cd2[4], cd2[5])
	c.p2 = getInt16(cd2[6], cd2[7])
	c.p3 = int8(cd2[8])
	c.p4 = getInt16(cd2[10], cd2[11])
	c.p5 = getInt16(cd2[12], cd2[13])
	c.p6 = int8(cd2[15])
	c.p7 = int8(cd2[14])
	c.p8 = getInt16(cd2[18], cd2[19])
	c.p9 = getInt16(cd2[20], cd2[21])
	c.p10 = cd2[22]

	h1E2 := cd1[1] & 0b11110000
	h2E2 := cd1[1] & 0b1111
	c.h1 = getUInt16(h1E2, cd1[2])
	c.h2 = getUInt16(h2E2, cd1[0])
	c.h3 = int8(cd1[3])
	c.h4 = int8(cd1[4])
	c.h5 = int8(cd1[5])
	c.h6 = cd1[6]
	c.h7 = int8(cd1[7])

	c.g1 = int8(cd1[12])
	c.g2 = getInt16(cd1[10], cd1[11])
	c.g3 = int8(cd1[13])

	return c
}

type calibration680 struct {
	t1 uint16
	t2 int16
	t3 int8

	p1     uint16
	p2     int16
	p3     int8
	p4, p5 int16
	p6, p7 int8
	p8, p9 int16
	p10    uint8

	h1, h2         uint16
	h6             uint8
	h3, h4, h5, h7 int8

	g1 int8
	g2 int16
	g3 int8
}

// compensateTemp returns temperature in °C, resolution is 0.01 °C.
// Output value of 5123 equals 51.23 C.
//
// temp_adc has to 16-20 bits of resolution.
func (c *calibration680) compensateTemp(tempADC uint32) (tempComp, tempFine float64) {
	var var1, var2 float64

	tempFloat := float64(tempADC)
	var1 = ((tempFloat / 16384.0) - (float64(c.t1) / 1024.0)) * float64(c.t2)
	var2 = ((tempFloat / 131072.0) - (float64(c.t1)/8192.0)*((tempFloat/131072.0)-(float64(c.t1)/8192.0))) * (float64(c.t3) * 16.0)
	tempFine = var1 + var2
	tempComp = tempFine / 5120.0
	return tempComp, tempFine
}

// compensatePressure returns pressure in Pa in Q24.8 format (24 integer
// bits and 8 fractional bits). Output value of 24674867 represents
// 24674867/256 = 96386.2 Pa = 963.862 hPa.
//
// raw has 20 bits of resolution.
func (c *calibration680) compensatePressure(pressureRaw uint32, tFine float64) (pressComp float64) {
	var var1, var2, var3 float64

	var1 = (tFine / 2.0) - 64000.0
	var2 = var1 * var1 * (float64(c.p6) / 131072.0)
	var2 = var2 + (var1 * float64(c.p5) * 2.0)
	var2 = (var2 / 4.0) + (float64(c.p4) * 65536.0)
	var1 = (((float64(c.p3) * var1 * var1) / 16384.0) + (float64(c.p2) * var1)) / 524288.0
	var1 = (1.0 + (var1 / 32768.0)) * float64(c.p1)

	pressComp = 1048576.0 - float64(pressureRaw)
	pressComp = ((pressComp - (var2 / 4096.0)) * 6250.0) / var1

	var1 = (float64(c.p9) * pressComp * pressComp) / 2147483648.0
	var2 = pressComp * (float64(c.p8) / 32768.0)
	var3 = (pressComp / 256.0) * (pressComp / 256.0) * (pressComp / 256.0) * (float64(c.p10) / 131072.0)

	pressComp = pressComp + (var1+var2+var3+(float64(c.p7)*128.0))/16.0
	return pressComp
}

// compensateHumidity returns humidity in %RH in Q22.10 format (22 integer
// and 10 fractional bits). Output value of 47445 represents 47445/1024 =
// 46.333%
//
// raw has 16 bits of resolution.
func (c *calibration680) compensateHumidity(humidityRaw uint32, tempComp float64) (humidityComp float64) {
	var var1, var2, var3, var4 float64

	var1 = (float64(humidityRaw)) - ((float64(c.h1) * 16.0) + ((float64(c.h3) / 2.0) * tempComp))
	var2 = var1 * ((float64(c.h2) / 262144.0) * (1.0 + ((float64(c.h4) / 16384.0) * tempComp) + ((float64(c.h5) / 1048576.0) * tempComp * tempComp)))
	var3 = float64(c.h6) / 16384.0
	var4 = float64(c.h7) / 2097152.0

	humidityComp = var2 + ((var3 + (var4 * tempComp)) * var2 * var2)
	//if humidityComp > 100.0 {
	//	humidityComp = 100.0
	//} else if humidityComp < 0.0 {
	//	humidityComp = 0.0
	//}

	return humidityComp
}
