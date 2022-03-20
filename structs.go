package main

import (
	"time"
)

type SensorReading struct {
	Temperature float64 `json:"temperature"`
	Pressure    float64 `json:"pressure"`
	Humidity    float64 `json:"humidity"`
	//HumidityBME float64   `json:"humidityBME"`
	CO2        uint16    `json:"co2"`
	Updated    time.Time `json:"-"`
	UpdatedStr string    `json:"updated"`
}

func NewSensorReading(date time.Time) SensorReading {
	return SensorReading{
		Updated:    date,
		UpdatedStr: date.Format("2006-01-02 15:04:05"), // ISO 8601 without timezone
	}
}
