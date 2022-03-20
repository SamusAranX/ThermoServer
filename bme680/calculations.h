/**
* Copyright (c) 2021 Bosch Sensortec GmbH. All rights reserved.
*
* BSD-3-Clause
*
* Redistribution and use in source and binary forms, with or without
* modification, are permitted provided that the following conditions are met:
*
* 1. Redistributions of source code must retain the above copyright
*    notice, this list of conditions and the following disclaimer.
*
* 2. Redistributions in binary form must reproduce the above copyright
*    notice, this list of conditions and the following disclaimer in the
*    documentation and/or other materials provided with the distribution.
*
* 3. Neither the name of the copyright holder nor the names of its
*    contributors may be used to endorse or promote products derived from
*    this software without specific prior written permission.
*
* THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
* "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
* LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS
* FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE
* COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT,
* INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
* (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
* SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
* HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT,
* STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING
* IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
* POSSIBILITY OF SUCH DAMAGE.
*
*/

#include <stdint.h>
#include <stddef.h>

struct bme680_calib_data {
	/*! Variable to store calibrated humidity data */
	uint16_t h1;
	/*! Variable to store calibrated humidity data */
	uint16_t h2;
	/*! Variable to store calibrated humidity data */
	int8_t h3;
	/*! Variable to store calibrated humidity data */
	int8_t h4;
	/*! Variable to store calibrated humidity data */
	int8_t h5;
	/*! Variable to store calibrated humidity data */
	uint8_t h6;
	/*! Variable to store calibrated humidity data */
	int8_t h7;
	/*! Variable to store calibrated gas data */
	int8_t gh1;
	/*! Variable to store calibrated gas data */
	int16_t gh2;
	/*! Variable to store calibrated gas data */
	int8_t gh3;
	/*! Variable to store calibrated temperature data */
	uint16_t t1;
	/*! Variable to store calibrated temperature data */
	int16_t t2;
	/*! Variable to store calibrated temperature data */
	int8_t t3;
	/*! Variable to store calibrated pressure data */
	uint16_t p1;
	/*! Variable to store calibrated pressure data */
	int16_t p2;
	/*! Variable to store calibrated pressure data */
	int8_t p3;
	/*! Variable to store calibrated pressure data */
	int16_t p4;
	/*! Variable to store calibrated pressure data */
	int16_t p5;
	/*! Variable to store calibrated pressure data */
	int8_t p6;
	/*! Variable to store calibrated pressure data */
	int8_t p7;
	/*! Variable to store calibrated pressure data */
	int16_t p8;
	/*! Variable to store calibrated pressure data */
	int16_t p9;
	/*! Variable to store calibrated pressure data */
	uint8_t p10;
	/*! Variable to store t_fine size */
	float t_fine;
	/*! Variable to store heater resistance range */
	uint8_t res_heat_range;
	/*! Variable to store heater resistance value */
	int8_t res_heat_val;
	/*! Variable to store error range */
	int8_t range_sw_err;
};

/* This internal API is used to calculate the temperature value in float */
float calc_temperature(uint32_t temp_adc, struct bme680_calib_data *calib);

/* This internal API is used to calculate the pressure value in float */
float calc_pressure(uint32_t pres_adc, const struct bme680_calib_data *calib);

/* This internal API is used to calculate the humidity value in float */
float calc_humidity(uint16_t hum_adc, const struct bme680_calib_data *calib);