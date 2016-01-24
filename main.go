/*
 * Copyright (c) Clinton Freeman 2015
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy of this software and
 * associated documentation files (the "Software"), to deal in the Software without restriction,
 * including without limitation the rights to use, copy, modify, merge, publish, distribute,
 * sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all copies or
 * substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT
 * NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
 * NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
 * DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
 */

package main

import (
	"bufio"
	"flag"
	"github.com/akualab/dmx"
	"github.com/kidoman/embd"
	_ "github.com/kidoman/embd/host/all"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type HRMsg struct {
	HeartRate int  // The current heart rate as returned by the polar H7.
	Contact   bool // Does the polar H7 currently have skin contact?
}

func main() {
	f, err := os.OpenFile("WeatherMachine2.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return // Ah-oh. Unable to log to file.
	}
	defer f.Close()
	log.SetOutput(f)
	log.Printf("INFO: Starting WeatherMachine2")

	var configFile string
	flag.StringVar(&configFile, "configFile", "weather-machine.json", "The path to the configuration file")
	flag.Parse()

	config, err := loadConfiguration(configFile)
	if err != nil {
		log.Printf("INFO: Unable to open '%s', using default values", configFile)
	}

	// Connect and initalise our Raspberry Pi GPIO pins.
	err = embd.InitGPIO()
	if err != nil {
		log.Printf("ERROR: Unable to initalize the raspberry pi GPIO ports.")
	}
	defer embd.CloseGPIO()

	embd.SetDirection(config.GPIOPinFan, embd.Out)
	embd.SetDirection(config.GPIOPinPump, embd.Out)
	embd.SetDirection(config.GPIOPinLight, embd.Out)
	embd.DigitalWrite(config.GPIOPinFan, embd.Low)
	embd.DigitalWrite(config.GPIOPinPump, embd.Low)
	embd.DigitalWrite(config.GPIOPinLight, embd.Low)

	// Connect to the DMX controller.
	dmx, e := dmx.NewDMXConnection(config.SmokeAddress)
	if e != nil {
		log.Printf("ERROR: Unable to connect to the DMX interface.")
		return
	}
	defer dmx.Close()

	conf := make(chan Configuration)
	hrMsg := make(chan HRMsg) // Channel for receiving heart rate messages from the PolarH7.
	weatherMachine := WeatherMachine{make(chan bool), dmx, config}
	update := idle

	go pollHeartRateMonitor(config.HRMMacAddress, hrMsg)
	go updateConfiguration(conf, configFile)
	for {
		msg := <-hrMsg
		log.Printf("INFO C: %t HR: %d", msg.Contact, msg.HeartRate)

		select {
		case c := <-conf:
			weatherMachine.config = c
			// Use a new config within the weather machine if the configfile has been updated.
		default:
			// Don't need to do anything. Just don't block.
		}

		update = update(&weatherMachine, msg)
	}
}

func updateConfiguration(c chan Configuration, configFile string) {
	ticker := time.NewTicker(time.Second * 30).C

	for {
		select {
		case <-ticker:
			config, err := loadConfiguration(configFile)
			if err != nil {
				log.Printf("INFO: Unable to open '%s', using default values", configFile)
			}
			c <- config
		}
	}
}

// pollHeartRateMonitor reads from the bluetooth heartrate monitor at the address specified
// by deviceID. It puts heart rate reatings onto the hr channel.
func pollHeartRateMonitor(deviceID string, hr chan HRMsg) {
	for {
		cmd := exec.Command("./WeatherMachine2-hrm", "--deviceID", deviceID)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("ERROR: Unable to read from HRM.")
			return
		}

		scanner := bufio.NewScanner(stdout)
		go func() {
			for scanner.Scan() {
				msg := strings.Split(scanner.Text(), ",")
				c, err := strconv.Atoi(msg[0])
				if err != nil {
					log.Printf("ERROR: Unable to parse contact")
				}

				i, err := strconv.Atoi(msg[1])
				if err != nil {
					log.Printf("ERROR: Unable to parse HR")
				}

				hr <- HRMsg{i, c == 1}
			}
		}()

		if err := cmd.Start(); err != nil {
			log.Printf("ERROR: Unable to start HRM.")
			return
		}

		cmd.Wait()
	}
}
