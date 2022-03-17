package main

import (
	"periph.io/x/conn/v3/physic"
	"time"
)

const (
	HectoPascal = 100 * physic.Pascal
)

type SensorReading struct {
	Temperature float64   `json:"temperature"`
	Pressure    float64   `json:"pressure"`
	Humidity    float64   `json:"humidity"`
	CO2         uint16    `json:"co2"`
	Updated     time.Time `json:"-"`
	UpdatedStr  string    `json:"updated"`
}

func NewSensorReading(date time.Time) SensorReading {
	return SensorReading{
		Updated:    date,
		UpdatedStr: date.Format("2006-01-02 15:04:05"), // ISO 8601 without timezone
	}
}
