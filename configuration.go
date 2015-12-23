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

type Configuration struct {
	SmokeVolume int // The amount of smoke for the machine to generate 0 - none, 127 - full blast.
	DeltaTFan   int // The number of milliseconds to wait between turning the smoke machine on and engaging the fan.
	DeltaTPump  int // The number of milliseconds to wait between turning the smoke machine on and engaging the rain pump.
}

// loadConfiguration reads a JSON file from the location specified at configFile and creates a configuration
// struct from the contents. On error a default configuration object is returned.
func loadConfiguration(configFile string) (c Configuration, err error) {
	c = Configuration{20, 10, 20} // Create default configuration.

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
