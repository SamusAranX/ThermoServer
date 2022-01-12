package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	"periph.io/x/devices/v3/bmxx80"
	"periph.io/x/host/v3"
	"time"
)

type ProgramArgs struct {
	// Server Options
	Host string `short:"H" long:"host" default:"127.0.0.1" description:"IP to listen on"`
	Port uint16 `short:"P" long:"port" default:"27315" description:"Port to listen on"`

	// Sensor Options
	Interval    uint16 `short:"I" long:"interval" default:"5" description:"Interval between readings"`
	I2CDevice   string `short:"D" long:"i2cdev" description:"The used I2C device (default: auto)"`
	HasHumidity bool   `long:"humidity" description:"Enable this if the sensor has a humidity sensor"`
}

var (
	args ProgramArgs

	currentEnv     physic.Env
	currentReading SensorReading
)

func updateReading(ch <-chan physic.Env) {
	for env := range ch {
		log.Println("New reading")

		currentEnv = env
		currentReading = SensorReadingFromEnv(currentEnv, time.Now())
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

// deviceSetup returns the bus and the device. the caller has the responsibility to close the bus
func deviceSetup(i2cdev string) (i2c.BusCloser, *bmxx80.Dev) {
	if _, err := host.Init(); err != nil {
		log.Fatalf("Initialization failed: %v", err)
	}

	bus, err := i2creg.Open(i2cdev)
	if err != nil {
		log.Fatalf("Couldn't open I2C device: %v", err)
	}

	dev, err := bmxx80.NewI2C(bus, 0x76, &bmxx80.DefaultOpts)
	if err != nil {
		log.Fatalf("Couldn't initialize sensor: %v", err)
	}

	return bus, dev
}

func main() {
	args = ProgramArgs{}
	argParser := flags.NewParser(&args, flags.Default)

	_, err := argParser.Parse()
	if err != nil {
		log.Fatal("arg parse fail")
	}

	// Boring i2c setup (error handling happens in this function)
	bus, dev := deviceSetup(args.I2CDevice)
	defer bus.Close()

	// SenseContinuous will take one reading immediately before looping
	readingChannel, err := dev.SenseContinuous(5 * time.Second)
	if err != nil {
		log.Fatalf("Couldn't start taking readings: %v", err)
	}
	defer dev.Halt()

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

	addr := fmt.Sprintf("%s:%d", args.Host, args.Port)
	srv := &http.Server{
		Addr:         addr,
		ReadTimeout:  4 * time.Second,
		WriteTimeout: 4 * time.Second,
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
