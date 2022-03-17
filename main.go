package main

import (
	"ThermoServer/bme680"
	"context"
	"encoding/json"
	"fmt"
	"github.com/aldernero/scd4x"
	"github.com/gorilla/mux"
	"github.com/jessevdk/go-flags"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"periph.io/x/conn/v3/i2c"
	"periph.io/x/conn/v3/i2c/i2creg"
	"periph.io/x/conn/v3/physic"
	"periph.io/x/host/v3"
	"time"
)

type ProgramArgs struct {
	// Server Options
	Host string `short:"H" long:"host" default:"127.0.0.1" description:"IP to listen on"`
	Port uint16 `short:"P" long:"port" default:"27315" description:"Port to listen on"`

	// Sensor Options
	Interval  uint16 `short:"I" long:"interval" default:"5" description:"Interval between readings"`
	I2CDevice string `short:"D" long:"i2cdev" description:"The used I2C device (default: auto)"`
}

var (
	args ProgramArgs

	currentEnv     physic.Env
	currentReading SensorReading

	scdDev *scd4x.SCD4x
)

const (
	MIN_TIMEOUT_SECONDS = 2
)

func updateReading(ch <-chan physic.Env) {
	for env := range ch {
		log.Println("New readings")

		currentEnv = env
		scdData, err := scdDev.ReadMeasurement()
		if err != nil {
			fmt.Errorf("error while reading SCD4x data: %v\n", err)
		}

		// BME680
		reading := NewSensorReading(time.Now())
		reading.Temperature = env.Temperature.Celsius()
		reading.Pressure = float64(env.Pressure) / float64(HectoPascal)

		// SCD41
		reading.Humidity = scdData.Rh
		reading.CO2 = scdData.CO2

		currentReading = reading
	}
}

func getOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func setupI2CBus(i2cdev string) i2c.BusCloser {
	if _, err := host.Init(); err != nil {
		log.Fatalf("Initialization failed: %v", err)
	}

	bus, err := i2creg.Open(i2cdev)
	if err != nil {
		log.Fatalf("Couldn't open I2C device: %v", err)
	}

	return bus
}

// setupBMESensor returns the bus and the device. the caller has the responsibility to close the bus
func setupBMESensor(i2cBus i2c.BusCloser) *bme680.Dev {
	deviceOpts := bme680.Opts{
		Temperature: bme680.O4x,
		Pressure:    bme680.O4x,
		Humidity:    bme680.O4x,
		Filter:      bme680.F4,
	}

	dev, err := bme680.NewI2C(i2cBus, 0x76, &deviceOpts)
	if err != nil {
		log.Fatalf("Couldn't initialize sensor: %v", err)
	}

	return dev
}

func setupSCDSensor(i2cBus i2c.BusCloser) *scd4x.SCD4x {
	sensor, err := scd4x.SensorInit(i2cBus, false)
	if err != nil {
		log.Fatalln(err.Error())
	}

	fmt.Println("Initializing SCD4x…")
	if err := sensor.StopMeasurements(); err != nil {
		log.Fatalf("Error while trying to stop periodic measurements: %v\n", err)
	}
	if err := sensor.StartMeasurements(); err != nil {
		log.Fatalf("Error while trying to start periodic measurements: %v\n", err)
	}
	fmt.Println("Done")

	return sensor
}

func main() {
	args = ProgramArgs{}
	argParser := flags.NewParser(&args, flags.Default)

	_, err := argParser.Parse()
	if err != nil {
		log.Fatal("arg parse fail")
	}

	// Boring i2c setup (error handling happens in these functions)
	bus := setupI2CBus(args.I2CDevice)
	defer bus.Close()

	bmeDev := setupBMESensor(bus)

	// SenseContinuous will take one reading immediately before looping
	intervalDuration := time.Duration(args.Interval)
	readingChannel, err := bmeDev.SenseContinuous(intervalDuration * time.Second)
	if err != nil {
		log.Fatalf("Couldn't start taking readings: %v", err)
	}
	defer bmeDev.Halt()

	scdDev = setupSCDSensor(bus)
	defer scdDev.StopMeasurements()

	fmt.Println("Waking up in a second…")

	// give the sensors time to wake up
	time.Sleep(1 * time.Second)

	// Start background measurements
	go updateReading(readingChannel)

	r := mux.NewRouter()
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		jsonStr, err := json.Marshal(currentReading)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		_, err = fmt.Fprintf(w, string(jsonStr))
		if err != nil {
			log.Fatalf("Couldn't send response: %v\n", err)
		}
	})

	timeoutLen := max(MIN_TIMEOUT_SECONDS, int(args.Interval))

	addr := fmt.Sprintf("%s:%d", args.Host, args.Port)
	srv := &http.Server{
		Addr:         addr,
		ReadTimeout:  time.Duration(timeoutLen) * time.Second,
		WriteTimeout: time.Duration(timeoutLen) * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      r,
	}

	go func() {
		if args.Host == "0.0.0.0" {
			localIP := getOutboundIP() // resolve local IP for easier debugging
			log.Printf("Listening on %s:%d…\n", localIP.String(), args.Port)
		} else {
			log.Printf("Listening on %s…\n", addr)
		}

		err := srv.ListenAndServe()
		log.Printf("Shutdown (%v)\n", err)
	}()

	sigChan := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(sigChan, os.Interrupt)

	<-sigChan

	// Give the server a timeout period of 4 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait until the timeout deadline.
	_ = srv.Shutdown(ctx)
	os.Exit(0)
}
