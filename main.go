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
	"flag"
	"log"
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

	// Prototype installation powerup. Need to poll heart rate monitor and enable as
	// required and close when HR drops to 0.
	d := make(chan bool)
	go enableLightPulse(60, d)
	go enableSmoke(config, d)
	go enableFan(config, d)
	go enablePump(config, d)

	time.Sleep(time.Second * 10)
	close(d)
	time.Sleep(time.Second * 10)
}

func pollHeartRateMonitor(hr chan int) {
	// Push Heart rate readings into the channel.
}

func pulseLight() {
	log.Printf("INFO: Light on")
	time.Sleep(time.Millisecond * 500)
	log.Printf("INFO: Light off")
}

// enableLightPulse starts the light pulsing by the frequency defined by hr.  The light remains
// pulsing till being notified to stop on d.
func enableLightPulse(hr int, d chan bool) {
	// Perform the first heart beat straight away.
	pulseLight()

	dt := int(60000.0 / float32(hr))
	ticker := time.NewTicker(time.Millisecond * time.Duration(dt)).C

	// Sharp fixed length, pulse of light with variable off gap depending on HR.
	for {
		select {
		case <-ticker:
			pulseLight()
		case <-d:
			log.Printf("INFO: Light off")
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
		case <-d:
			log.Printf("INFO: Pump Off")
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
		case <-d:
			log.Printf("INFO: Fan Off")
			return
		}
	}
}

// enableSmoke enages the DMX smoke machine by the SmokeVolume amount in the configuration.
// Smoke Machine remains on till being notified to stop on d.
func enableSmoke(c Configuration, d chan bool) {
	log.Printf("INFO: Smoke on")
	<-d
	log.Printf("INFO: Smoke off")
}
