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
	//_ "github.com/kidoman/embd/host/all"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type HRMsg struct {
	HeartRate int
	Contact   bool
}

func main() {
	log.Printf("INFO: Starting Weather-Machine")
	var configFile string

	flag.StringVar(&configFile, "configFile", "weather-machine.json", "The path to the configuration file")
	flag.Parse()

	config, err := loadConfiguration(configFile)
	if err != nil {
		log.Printf("INFO: Unable to open '%s', using default values", configFile)
	}

	// Connect to the GPIO ports.
	embd.InitGPIO()
	defer embd.CloseGPIO()

	// Connect to the DMX controller.
	dmx, e := dmx.NewDMXConnection(config.SmokeAddress)
	if e != nil {
		log.Printf("ERROR: Unable to make DMX connection to smoke machine.")
		return
	}
	defer dmx.Close()

	embd.SetDirection(config.GPIOPinFan, embd.Out)
	embd.SetDirection(config.GPIOPinPump, embd.Out)
	embd.SetDirection(config.GPIOPinLight, embd.Out)

	// Make sure all our GPIO pins are off.
	embd.DigitalWrite(config.GPIOPinFan, embd.Low)
	embd.DigitalWrite(config.GPIOPinPump, embd.Low)
	embd.DigitalWrite(config.GPIOPinLight, embd.Low)

	// Prototype installation powerup. Need to poll heart rate monitor and enable as
	// required and close when HR drops to 0.
	d := make(chan bool)
	hrMsg := make(chan HRMsg)
	running := false

	go pollHeartRateMonitor(config.HRMMacAddress, hrMsg)
	for {
		msg := <-hrMsg
		log.Printf("C: %t HR: %d", msg.Contact, msg.HeartRate)

		// When someone touches the installation. Give instant feedback.
		if msg.Contact && !running {
			log.Printf("INFO: Light on")
			enableLight(config.S1Beat, dmx)
			//embd.DigitalWrite(config.GPIOPinLight, embd.Low)
		}

		if msg.Contact && msg.HeartRate > 0 && !running {
			go enableLightPulse(config, msg.HeartRate, d, dmx)
			go enableSmoke(config, d, dmx)
			go enableFan(config, d)
			go enablePump(config, d)
			running = true
		} else if !msg.Contact && running {
			// Stop all four elements of the installation.
			d <- true
			d <- true
			d <- true
			d <- true
			running = false
		}

		// When someone lets go of the installation. Give instant feedback.
		if !msg.Contact && !running {
			log.Printf("INFO: Light off")
			disableLight(dmx)
			//embd.DigitalWrite(config.GPIOPinLight, embd.High)
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

func enableLight(l LightColour, dmx *dmx.DMX) {
	dmx.SetChannel(4, byte(l.Red))
	dmx.SetChannel(5, byte(l.Green))
	dmx.SetChannel(6, byte(l.Blue))
	dmx.SetChannel(7, byte(l.Amber))
	dmx.SetChannel(8, byte(l.Dimmer))
	dmx.Render()
}

func disableLight(dmx *dmx.DMX) {
	dmx.SetChannel(4, 0)
	dmx.SetChannel(5, 0)
	dmx.SetChannel(6, 0)
	dmx.SetChannel(7, 0)
	dmx.SetChannel(8, 0)
	dmx.Render()
}

// pulseLight pulses the light for a fixed duration.
func pulseLight(c Configuration, dmx *dmx.DMX) {
	log.Printf("INFO: Light on")
	enableLight(c.S1Beat, dmx)
	time.Sleep(time.Millisecond * time.Duration(c.S1Duration))
	disableLight(dmx)

	enableLight(c.S2Beat, dmx)
	time.Sleep(time.Millisecond * time.Duration(c.S2Duration))
	disableLight(dmx)

	//embd.DigitalWrite(c.GPIOPinLight, embd.Low)
	//embd.DigitalWrite(c.GPIOPinLight, embd.High)
	log.Printf("INFO: Light off")
}

// enableLightPulse starts the light pulsing by the frequency defined by hr. The light remains
// pulsing till being notified to stop on d.
func enableLightPulse(c Configuration, hr int, d chan bool, dmx *dmx.DMX) {
	// Perform the first heart beat straight away.
	pulseLight(c, dmx)

	dt := int((60000.0 / float32(hr)) * c.BeatRate)
	ticker := time.NewTicker(time.Millisecond * time.Duration(dt)).C

	// Sharp fixed length, pulse of light with variable off gap depending on HR.
	for {
		select {
		case <-ticker:
			pulseLight(c, dmx)
		case <-d:
			return
		}
	}
}

// enablePump switches the relay on for the water pump after DeltaTPump milliseconds have expired
// in the configuration.  Pump remains on till being notified to stop on d.
func enablePump(c Configuration, d chan bool) {
	dt := time.NewTimer(time.Millisecond * time.Duration(c.DeltaTPump)).C

	for {
		select {
		case <-dt:
			log.Printf("INFO: Pump on")
			embd.DigitalWrite(c.GPIOPinPump, embd.Low)
		case <-d:
			log.Printf("INFO: Pump Off")
			embd.DigitalWrite(c.GPIOPinPump, embd.High)
			return
		}
	}
}

// enableFan switches the relay on for the fan after DeltaTFan milliseconds have expired
// in the configuration.  Pump remains on till being notified to stop on d.
func enableFan(c Configuration, d chan bool) {
	dt := time.NewTimer(time.Millisecond * time.Duration(c.DeltaTFan)).C

	for {
		select {
		case <-dt:
			log.Printf("INFO: Fan On")
			embd.DigitalWrite(c.GPIOPinFan, embd.Low)
		case <-d:
			log.Printf("INFO: Fan Off")
			// Wait for the fan duration to clear the smoke chamber.
			ft := time.NewTimer(time.Millisecond * time.Duration(c.FanDuration)).C
			<-ft
			embd.DigitalWrite(c.GPIOPinFan, embd.High)
			return
		}
	}
}

// enableSmoke enages the DMX smoke machine by the SmokeVolume amount in the configuration.
// Smoke Machine remains on till being notified to stop on d.
func enableSmoke(c Configuration, d chan bool, dmx *dmx.DMX) {
	dt := time.NewTimer(time.Millisecond * time.Duration(c.DeltaTSmoke)).C

	for {
		select {
		case <-dt:
			log.Printf("INFO: Smoke on")
			dmx.SetChannel(1, byte(c.SmokeVolume))
			dmx.Render()
			st := time.NewTimer(time.Millisecond * time.Duration(c.SmokeDuration)).C
			<-st
			log.Printf("INFO: Smoke off")
			dmx.SetChannel(1, 0)
			dmx.Render()

		case <-d:
			log.Printf("INFO: Smoke off")
			dmx.SetChannel(1, 0)
			dmx.Render()
			return
		}
	}
}
