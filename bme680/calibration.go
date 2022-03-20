// Copyright 2016 The Periph Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package bme680

// #include "calculations.h"
import "C"

// newCalibration parses calibration data from both buffers.
func newCalibration(cd1, cd2 []byte) (calib calibrationData) {
	// cd1 covers 0xE1 through 0xEE
	// cd2 covers 0x8A through 0xA0

	getShort := func(lsb, msb byte) C.short {
		return C.short(int16(lsb) | (int16(msb) << 8))
	}

	getUShort := func(lsb, msb byte) C.ushort {
		return C.ushort(uint16(lsb) | (uint16(msb) << 8))
	}

	h1E2 := cd1[1] & 0b11110000
	h2E2 := cd1[1] & 0b1111
	calib.c.h1 = getUShort(h1E2, cd1[2])
	calib.c.h2 = getUShort(h2E2, cd1[0])
	calib.c.h3 = C.schar(cd1[3])
	calib.c.h4 = C.schar(cd1[4])
	calib.c.h5 = C.schar(cd1[5])
	calib.c.h6 = C.uchar(cd1[6])
	calib.c.h7 = C.schar(cd1[7])

	calib.c.gh1 = C.schar(cd1[12])
	calib.c.gh2 = getShort(cd1[10], cd1[11])
	calib.c.gh3 = C.schar(cd1[13])

	calib.c.t1 = getUShort(cd1[8], cd1[9])
	calib.c.t2 = getShort(cd2[0], cd2[1])
	calib.c.t3 = C.schar(cd2[2])

	calib.c.p1 = getUShort(cd2[4], cd2[5])
	calib.c.p2 = getShort(cd2[6], cd2[7])
	calib.c.p3 = C.schar(cd2[8])
	calib.c.p4 = getShort(cd2[10], cd2[11])
	calib.c.p5 = getShort(cd2[12], cd2[13])
	calib.c.p6 = C.schar(cd2[15])
	calib.c.p7 = C.schar(cd2[14])
	calib.c.p8 = getShort(cd2[18], cd2[19])
	calib.c.p9 = getShort(cd2[20], cd2[21])
	calib.c.p10 = C.uchar(cd2[22])

	return calib
}

// CalibrationData translates bme680_calib_data to Go
type calibrationData struct {
	c C.struct_bme680_calib_data
}

// compensateTemp returns temperature in °C.
func (c *calibrationData) compensateTemp(tempRaw uint32) (tempComp float32) {
	CUintTempRaw := C.uint(tempRaw)
	CFloatTempComp := C.calc_temperature(CUintTempRaw, &c.c)
	tempComp = float32(CFloatTempComp)
	return tempComp
}

// compensatePressure returns pressure in Pascal.
func (c calibrationData) compensatePressure(pressureRaw uint32) (pressureComp float32) {
	CUintPressureRaw := C.uint(pressureRaw)
	CFloatPressureComp := C.calc_pressure(CUintPressureRaw, &c.c)
	pressureComp = float32(CFloatPressureComp) / 100
	return pressureComp
}

// compensateHumidity returns humidity in %RH.
func (c calibrationData) compensateHumidity(humidityRaw uint16) (humidityComp float32) {
	CUshortHumidityRaw := C.ushort(humidityRaw)
	CFloatHumidityComp := C.calc_humidity(CUshortHumidityRaw, &c.c)
	humidityComp = float32(CFloatHumidityComp)
	return humidityComp
}
