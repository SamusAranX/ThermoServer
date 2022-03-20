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

#include "calculations.h"

float calc_temperature(uint32_t temp_adc, struct bme680_calib_data *calib)
{
	float var1 = 0;
	float var2 = 0;
	float calc_temp = 0;

	/* calculate var1 data */
	var1  = ((((float)temp_adc / 16384.0f) - ((float)calib->t1 / 1024.0f)) * ((float)calib->t2));

	/* calculate var2 data */
	var2  = (((((float)temp_adc / 131072.0f) - ((float)calib->t1 / 8192.0f)) *
		(((float)temp_adc / 131072.0f) - ((float)calib->t1 / 8192.0f))) *
		((float)calib->t3 * 16.0f));

	/* t_fine value */
	calib->t_fine = (var1 + var2);

	/* compensated temperature data */
	calc_temp = ((calib->t_fine) / 5120.0f);

	return calc_temp;
}

float calc_pressure(uint32_t pres_adc, const struct bme680_calib_data *calib)
{
	float var1 = 0;
	float var2 = 0;
	float var3 = 0;
	float calc_pres = 0;

	var1 = (((float)calib->t_fine / 2.0f) - 64000.0f);
	var2 = var1 * var1 * (((float)calib->p6) / (131072.0f));
	var2 = var2 + (var1 * ((float)calib->p5) * 2.0f);
	var2 = (var2 / 4.0f) + (((float)calib->p4) * 65536.0f);
	var1 = (((((float)calib->p3 * var1 * var1) / 16384.0f) + ((float)calib->p2 * var1)) / 524288.0f);
	var1 = ((1.0f + (var1 / 32768.0f)) * ((float)calib->p1));
	calc_pres = (1048576.0f - ((float)pres_adc));

	/* Avoid exception caused by division by zero */
	if ((int)var1 != 0) {
		calc_pres = (((calc_pres - (var2 / 4096.0f)) * 6250.0f) / var1);
		var1 = (((float)calib->p9) * calc_pres * calc_pres) / 2147483648.0f;
		var2 = calc_pres * (((float)calib->p8) / 32768.0f);
		var3 = ((calc_pres / 256.0f) * (calc_pres / 256.0f) * (calc_pres / 256.0f) * (calib->p10 / 131072.0f));
		calc_pres = (calc_pres + (var1 + var2 + var3 + ((float)calib->p7 * 128.0f)) / 16.0f);
	} else {
		calc_pres = 0;
	}

	return calc_pres;
}

float calc_humidity(uint16_t hum_adc, const struct bme680_calib_data *calib)
{
	float calc_hum = 0;
	float var1 = 0;
	float var2 = 0;
	float var3 = 0;
	float var4 = 0;
	float temp_comp;

	/* compensated temperature data */
	temp_comp  = ((calib->t_fine) / 5120.0f);

	var1 = (float)((float)hum_adc) - (((float)calib->h1 * 16.0f) + (((float)calib->h3 / 2.0f) * temp_comp));

	var2 = var1 * ((float)(((float) calib->h2 / 262144.0f) * (1.0f + (((float)calib->h4 / 16384.0f)
		* temp_comp) + (((float)calib->h5 / 1048576.0f) * temp_comp * temp_comp))));

	var3 = (float) calib->h6 / 16384.0f;

	var4 = (float) calib->h7 / 2097152.0f;

	calc_hum = var2 + ((var3 + (var4 * temp_comp)) * var2 * var2);

	if (calc_hum > 100.0f)
		calc_hum = 100.0f;
	else if (calc_hum < 0.0f)
		calc_hum = 0.0f;

	return calc_hum;
}