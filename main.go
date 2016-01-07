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
	"os/exec"
	"strconv"
	"time"
)

func main() {
	log.Printf("INFO: Starting Weather-Machine")
	var configFile string

	flag.StringVar(&configFile, "configFile", "weather-machine.json", "The path to the configuration file")
	flag.Parse()

	config, err := loadConfiguration(configFile)
	if err != nil {
		log.Printf("INFO: Unable to open '%s', using default values", configFile)
	}

	embd.InitGPIO()
	defer embd.CloseGPIO()

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
	hr := make(chan int)
	running := false

	go pollHeartRateMonitor(config.HRMMacAddress, hr)
	for {
		heartRate := <-hr
		log.Printf("HR: heartRate %d", heartRate)

		// If we have just paired with the HRM, someone has touched the paddles.
		if heartRate == -1 {
			embd.DigitalWrite(config.GPIOPinLight, embd.High)
		}

		if heartRate > 0 && !running {
			log.Printf("ENABLE INSTALLATION")
			go enableLightPulse(config, heartRate, d)
			go enableSmoke(config, d)
			go enableFan(config, d)
			go enablePump(config, d)
			running = true
		} else if heartRate == 0 && running {
			log.Printf("DISABLE INSTALLATION")
			// Stop all four elements of the installation.
			d <- true
			d <- true
			d <- true
			d <- true
			running = false
		}
	}
}

// pollHeartRateMonitor reads from the bluetooth heartrate monitor at the address specified
// by deviceID. It puts heart rate reatings onto the hr channel.
func pollHeartRateMonitor(deviceID string, hr chan int) {
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
				i, err := strconv.Atoi(scanner.Text())
				if err != nil {
					log.Printf("ERROR: Unable to parse HR")
				}

				hr <- i
			}
		}()

		if err := cmd.Start(); err != nil {
			log.Printf("ERROR: Unable to start HRM.")
			return
		}

		cmd.Wait()
	}
}

// pulseLight pulses the light for a fixed duration.
func pulseLight(c Configuration) {
	log.Printf("INFO: Light on")
	embd.DigitalWrite(c.GPIOPinLight, embd.High)
	time.Sleep(time.Millisecond * 500)
	embd.DigitalWrite(c.GPIOPinLight, embd.Low)
	log.Printf("INFO: Light off")
}

// enableLightPulse starts the light pulsing by the frequency defined by hr. The light remains
// pulsing till being notified to stop on d.
func enableLightPulse(c Configuration, hr int, d chan bool) {
	// Perform the first heart beat straight away.
	pulseLight(c)

	dt := int(60000.0 / float32(hr))
	ticker := time.NewTicker(time.Millisecond * time.Duration(dt)).C

	// Sharp fixed length, pulse of light with variable off gap depending on HR.
	for {
		select {
		case <-ticker:
			pulseLight(c)
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
			embd.DigitalWrite(c.GPIOPinPump, embd.High)
		case <-d:
			log.Printf("INFO: Pump Off")
			embd.DigitalWrite(c.GPIOPinPump, embd.Low)
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
			embd.DigitalWrite(c.GPIOPinFan, embd.High)
		case <-d:
			log.Printf("INFO: Fan Off")
			embd.DigitalWrite(c.GPIOPinFan, embd.Low)
			return
		}
	}
}

// enableSmoke enages the DMX smoke machine by the SmokeVolume amount in the configuration.
// Smoke Machine remains on till being notified to stop on d.
func enableSmoke(c Configuration, d chan bool) {
	dt := time.NewTimer(time.Millisecond * time.Duration(c.DeltaTSmoke)).C
	dmx, e := dmx.NewDMXConnection(c.SmokeAddress)
	if e != nil {
		log.Printf("Unable to connect to smoke machine.")
	}
	defer dmx.Close()

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
