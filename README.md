# ThermoServer

This program was written for a Raspberry Pi using Pimoroni's BMP280 breakout.
Chances are it'll also work with the BME280 and BME680, but those are untested.

## Usage

On the server machine:
```shell
$ go get

$ go build -o thermoserver

$ ./thermoserver -H 0.0.0.0
2022/01/13 00:40:06 New reading
2022/01/13 00:40:06 Listening on <server IP>:27315…
```

From another machine on the network:
```shell
$ curl <server IP>:27315
{"temperature":18.18,"pressure":1018.5345703125,"updated":"2022-01-13 00:40:16"}
```