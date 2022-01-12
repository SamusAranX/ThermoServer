package main

import (
	"periph.io/x/conn/v3/physic"
	"time"
)

const (
	HectoPascal physic.Pressure = 100 * physic.Pascal
)

type SensorReading struct {
	Temperature float64   `json:"temperature"`
	Pressure    float64   `json:"pressure"`
	Humidity    *float64  `json:"humidity,omitempty"`
	Updated     time.Time `json:"-"`
	UpdatedStr  string    `json:"updated"`
}

func SensorReadingFromEnv(env physic.Env, date time.Time) SensorReading {
	sr := SensorReading{
		Temperature: env.Temperature.Celsius(),
		Pressure:    float64(env.Pressure) / float64(HectoPascal),
		Updated:     date,
		UpdatedStr:  date.Format("2006-01-02 15:04:05"), // ISO 8601 without timezone
	}

	if args.HasHumidity {
		newHumidity := float64(env.Humidity / physic.PercentRH)
		sr.Humidity = &newHumidity
	}

	return sr
}
