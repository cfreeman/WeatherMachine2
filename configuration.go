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
	"encoding/json"
	"os"
)

type LightColour struct {
	Red    int // The intensity of the red channel. (1-255)
	Green  int // The intensity of the green channel. (1-255)
	Blue   int // The intensity of the blue channel. (1-255)
	Amber  int // The intensity of the amber channel. (1-255)
	Dimmer int // The intensity of the dimmer channel. (1-255)
}

type Configuration struct {
	SmokeVolume   int         // The amount of smoke for the machine to generate 0 - none, 127 - full blast.
	DeltaTSmoke   int         // The number of milliseconds to wait before turning the smoke machine on.
	DeltaTFan     int         // The number of milliseconds to wait before engaging the fan.
	DeltaTPump    int         // The number of milliseconds to wait before and engaging the rain pump.
	HRMMacAddress string      // The bluetooth peripheral ID for the heart rate monitor.
	GPIOPinFan    int         // The GPIO pin id to use for controlling the fan.
	GPIOPinPump   int         // The GPIO pin id to use for controlling the pump.
	SmokeAddress  string      // The serial address of the DMX controller for the smoke machine.
	SmokeDuration int         // The number of milliseconds to activate the smoke machine.
	FanDuration   int         // The number of milliseconds to leave the fan running.
	BeatRate      float32     // Heartrate scale. 0.0 -> nothing. 1.0 full heartrate.
	S1Beat        LightColour // The colour to use for the first beat of the heart.
	S1Duration    int         // The number of milliseconds to leave the light on for the first heart beat.
	S2Beat        LightColour // The colour to use for the second (S2) beat of the heart.
	S2Duration    int         // The number of milliseconds to leave the light on for the second heart beat.
	S1Pause       int         // The number of milliseconds to pause between S1 and S2.
	SmokeInterval int         // The number of milliseconds to wait before puffing smoke.
}

// loadConfiguration reads a JSON file from the location specified at configFile and creates a configuration
// struct from the contents. On error a default configuration object is returned.
func loadConfiguration(configFile string) (c Configuration, err error) {
	c = Configuration{63, 10, 20, 30, "00:22:D0:97:C4:C0", 16, 20, "/dev/ttyUSB0", 500, 500, 0.9, LightColour{200, 10, 10, 50, 155}, 500, LightColour{200, 10, 10, 50, 50}, 50, 50, 1000} // Create default configuration.

	file, err := os.Open(configFile)
	if err != nil {
		return c, err
	}
	defer file.Close()

	// Parse JSON from the configuration file.
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&c)
	return c, err
}
