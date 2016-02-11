/*
 * Copyright (c) Clinton Freeman 2016
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
	"github.com/akualab/dmx"
	"github.com/kidoman/embd"
	_ "github.com/kidoman/embd/host/all"
	"log"
	"time"
)

// WeatherMachine holds connections to everything we need to manipulate the installation.
type WeatherMachine struct {
	stop   chan bool     // Channel for stopping the control elements of the installation.
	dmx    *dmx.DMX      // The DMX connection for writting messages to the Smoke machine and lights.
	config Configuration // The configuration element for the installation.
}

// ****************************************************************************
// ****************************************************************************
// Functions for manipulating the installation state; idle, warmup and running.
// ****************************************************************************
// ****************************************************************************

// stateFunctions are used to manipulate the WeatherMachine through the various states.
type stateFn func(state *WeatherMachine, msg HRMsg) stateFn

// idle is the state the weathermachine enters when sitting alone, with no one interacting with it.
func idle(state *WeatherMachine, msg HRMsg) (sF stateFn) {
	if msg.Contact {
		log.Printf("INFO: entering warmup")
		enableLight(state.config.S1Beat, state.config, state.dmx)
		return warmup // skin contact has been made, enable light and enter warmup.
	}

	return idle // remain idle.
}

// warmup is the state the weathermachine enters when someone first touches it.
func warmup(state *WeatherMachine, msg HRMsg) stateFn {
	if msg.Contact && msg.HeartRate > 0 {
		go enableLightPulse(state.config, msg.HeartRate, state.stop, state.dmx)
		go enableSmoke(state.config, state.stop, state.dmx)
		go enableFan(state.config, state.stop)
		go enablePump(state.config, state.stop)

		log.Printf("INFO: entering running")
		return running // skin contact and heart rate recieved, start the installation.
	} else if !msg.Contact {
		disableLight(state.config, state.dmx)

		log.Printf("INFO: entering idle")
		return idle // skin contact lost. Return to idle.
	}

	return warmup
}

// running is the state the weathermachine enters when someone is engaging with it.
func running(state *WeatherMachine, msg HRMsg) stateFn {
	if !msg.Contact {
		state.stop <- true
		state.stop <- true
		state.stop <- true
		state.stop <- true

		log.Printf("INFO: entering idle")
		return idle // skin contact lost. Return to idle.
	}

	return running // Keep the installation running.
}

// ****************************************************************************
// ****************************************************************************
// Functions for manipulating the physical installation; lights, smoke and fan.
// ****************************************************************************
// ****************************************************************************

// enableLight turns on the light via the supplied DMX connection 'dmx' with the supplied colour 'l'.
func enableLight(l LightColour, c Configuration, dmx *dmx.DMX) {
	log.Printf("INFO: Light on")
	embd.DigitalWrite(c.GPIOPinLight, embd.High)
	dmx.SetChannel(4, byte(l.Red))
	dmx.SetChannel(5, byte(l.Green))
	dmx.SetChannel(6, byte(l.Blue))
	dmx.SetChannel(7, byte(l.Amber))
	dmx.SetChannel(8, byte(l.Dimmer))
	dmx.Render()
}

// disableLight turns off the light via the supplied DMX connection 'dmx'.
func disableLight(c Configuration, dmx *dmx.DMX) {
	log.Printf("INFO: Light off")
	embd.DigitalWrite(c.GPIOPinLight, embd.Low)
	dmx.SetChannel(4, 0)
	dmx.SetChannel(5, 0)
	dmx.SetChannel(6, 0)
	dmx.SetChannel(7, 0)
	dmx.SetChannel(8, 0)
	dmx.Render()
}

// pulseLight pulses the light for a fixed duration.
func pulseLight(c Configuration, dmx *dmx.DMX) {
	enableLight(c.S1Beat, c, dmx)
	time.Sleep(time.Millisecond * time.Duration(c.S1Duration))
	disableLight(c, dmx)

	time.Sleep(time.Millisecond * time.Duration(c.S1Pause))

	enableLight(c.S2Beat, c, dmx)
	time.Sleep(time.Millisecond * time.Duration(c.S2Duration))
	disableLight(c, dmx)
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

// pulsePump runs the pump for the duration specified in the configuration.
func pulsePump(c Configuration) {
	log.Printf("INFO: Pump on")
	embd.DigitalWrite(c.GPIOPinPump, embd.High)

	time.Sleep(time.Millisecond * time.Duration(c.PumpDuration))

	log.Printf("INFO: Pump Off")
	embd.DigitalWrite(c.GPIOPinPump, embd.Low)
}

// enablePump switches the relay on for the water pump after DeltaTPump milliseconds have expired
// in the configuration. Pump remains on till being notified to stop on d.
func enablePump(c Configuration, d chan bool) {
	dt := time.NewTimer(time.Millisecond * time.Duration(c.DeltaTPump)).C
	var ticker <-chan time.Time

	for {
		select {
		case <-dt:
			pulsePump(c)
			ticker = time.NewTicker(time.Millisecond * time.Duration(c.PumpInterval)).C

		case <-ticker:
			pulsePump(c)

		case <-d:
			return
		}
	}
}

// enableFan switches the relay on for the fan after DeltaTFan milliseconds have expired
// in the configuration. Fan remains on till being notified to stop on d.
func enableFan(c Configuration, d chan bool) {
	dt := time.NewTimer(time.Millisecond * time.Duration(c.DeltaTFan)).C

	for {
		select {
		case <-dt:
			log.Printf("INFO: Fan On")
			embd.DigitalWrite(c.GPIOPinFan, embd.High)

		case <-d:
			log.Printf("INFO: Fan Off")
			// Wait for the fan duration to clear the smoke chamber.
			ft := time.NewTimer(time.Millisecond * time.Duration(c.FanDuration)).C
			<-ft
			embd.DigitalWrite(c.GPIOPinFan, embd.Low)
			return
		}
	}
}

// puffSmoke enables the smoke machine via the supplied DMX connection 'dmx' for a period of
// time and intentsity supplied in configuration.
func puffSmoke(c Configuration, dmx *dmx.DMX) {
	log.Printf("INFO: Smoke on")
	dmx.SetChannel(1, byte(c.SmokeVolume))
	dmx.Render()

	time.Sleep(time.Millisecond * time.Duration(c.SmokeDuration))

	log.Printf("INFO: Smoke off")
	dmx.SetChannel(1, 0)
	dmx.Render()
}

// enableSmoke enages the DMX smoke machine by the SmokeVolume amount in the configuration.
// Smoke Machine remains on till being notified to stop on d.
func enableSmoke(c Configuration, d chan bool, dmx *dmx.DMX) {
	dt := time.NewTimer(time.Millisecond * time.Duration(c.DeltaTSmoke)).C
	var ticker <-chan time.Time

	for {
		select {
		case <-dt:
			puffSmoke(c, dmx)
			ticker = time.NewTicker(time.Millisecond * time.Duration(c.SmokeInterval)).C

		case <-ticker:
			puffSmoke(c, dmx)

		case <-d:
			return
		}
	}
}
