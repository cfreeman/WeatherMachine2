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

// I2C
type RelayControl struct {
	bus     embd.I2CBus
	address byte
	mode    byte
	regData byte
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

	// Connect and initalise Raspberry Pi I2C
	err = embd.InitI2C()
	if err != nil {
		log.Printf("ERROR: Unable to initalize the Raspberry Pi I2C. Ensure you have configured the PI I2C ports")
	}
	defer embd.CloseI2C()
	bus := embd.NewI2CBus(1)

	// Create relay controller
	relayCtrl := NewRelayCtrl(bus)

	// Reset relay
	relayCtrl.bus.WriteByteToReg(relayCtrl.address, relayCtrl.mode, relayCtrl.regData)

	// If we don't have the address of a heart rate monitor. Look for it.
	if strings.Compare(config.HRMMacAddress, "0") == 0 {
		log.Printf("INFO: Scanning for HRM.")
		config.HRMMacAddress = scanHeartRateMonitor()
		log.Printf("INFO: Found %s\n", config.HRMMacAddress)
		saveConfiguration(configFile, config)
	}

	// Connect to the DMX controller.
	dmx, e := dmx.NewDMXConnection(config.SmokeAddress)
	if e != nil {
		log.Printf("ERROR: Unable to connect to the DMX interface.")
		return
	}
	defer dmx.Close()

	conf := make(chan Configuration)
	hrMsg := make(chan HRMsg) // Channel for receiving heart rate messages from the PolarH7.
	weatherMachine := WeatherMachine{make(chan bool), dmx, config, time.Now().Add(-time.Duration(config.FanDuration)), relayCtrl}
	update := idle

	go pollHeartRateMonitor(config.HRMMacAddress, hrMsg)
	go updateConfiguration(conf, configFile)
	for {
		msg := <-hrMsg

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

func NewRelayCtrl(bus embd.I2CBus) *RelayControl {
	return &RelayControl{bus: bus, address: 0x20, mode: 0x06, regData: 0xff}
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

func scanHeartRateMonitor() string {
	cmd := exec.Command("./WeatherMachine2-scan")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("ERROR: Unable to scan HRM.")
		return "0"
	}

	scanner := bufio.NewScanner(stdout)
	id := "0"
	go func() {
		for scanner.Scan() {
			id = scanner.Text()
			return
		}
	}()

	if err := cmd.Start(); err != nil {
		log.Printf("ERROR: Unable to start scanning HRM.")
		return id
	}

	cmd.Wait()
	return id
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
