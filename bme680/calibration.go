package bme680

type calibrationData struct {
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

	h1, h2     uint16
	h3, h4, h5 int8
	h6         uint8
	h7         int8

	g1 int8
	g2 int16
	g3 int8
}

// newCalibration parses calibration data from both buffers.
func newCalibration(cd1, cd2 []byte) (c calibrationData) {
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

// compensateTemp returns temperature in °C.
func (c calibrationData) compensateTemp(tempRaw uint32) (tempComp, tempFine float64) {
	var var1, var2 float64

	tempFloat := float64(tempRaw)
	var1 = ((tempFloat / 16384.0) - (float64(c.t1) / 1024.0)) * float64(c.t2)
	var2 = ((tempFloat / 131072.0) - (float64(c.t1)/8192.0)*((tempFloat/131072.0)-(float64(c.t1)/8192.0))) * (float64(c.t3) * 16.0)
	tempFine = var1 + var2
	tempComp = tempFine / 5120.0
	return tempComp, tempFine
}

// compensatePressure returns pressure in Pascal.
func (c calibrationData) compensatePressure(pressureRaw uint32, tFine float64) (pressComp float64) {
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
	return pressComp / 100 // divide by 100 to get pascal directly
}

// compensateHumidity returns humidity in %RH.
func (c calibrationData) compensateHumidity(humidityRaw uint16, tFine float64) (humidityComp float64) {
	var var1, var2, var3, var4 float64

	tempComp := tFine / 5120

	var1 = (float64(humidityRaw)) - ((float64(c.h1) * 16.0) + ((float64(c.h3) / 2.0) * tempComp))
	var2 = var1 * ((float64(c.h2) / 262144.0) * (1.0 + ((float64(c.h4) / 16384.0) * tempComp) + ((float64(c.h5) / 1048576.0) * tempComp * tempComp)))
	var3 = float64(c.h6) / 16384.0
	var4 = float64(c.h7) / 2097152.0

	humidityComp = var2 + ((var3 + (var4 * tempComp)) * var2 * var2)
	if humidityComp > 100.0 {
		humidityComp = 100.0
	} else if humidityComp < 0.0 {
		humidityComp = 0.0
	}

	return humidityComp
}
