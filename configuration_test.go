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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestConfiguration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Configuration Suite")
}

var _ = Describe("Configuration", func() {
	Context("loading", func() {
		It("should throw an error for an invalid config file", func() {
			c, err := loadConfiguration("foo")

			Ω(err).ShouldNot(BeNil())
			Ω(c.SmokeVolume).Should(Equal(63))
			Ω(c.DeltaTSmoke).Should(Equal(10))
			Ω(c.DeltaTFan).Should(Equal(20))
			Ω(c.DeltaTPump).Should(Equal(30))
			Ω(c.HRMMacAddress).Should(Equal("00:22:D0:97:C4:C0"))
			Ω(c.GPIOPinFan).Should(Equal(16))
			Ω(c.GPIOPinPump).Should(Equal(20))
		})

		It("should be able to load a valid config file", func() {
			c, err := loadConfiguration("testdata/test-config.json")

			Ω(err).Should(BeNil())
			Ω(c.SmokeVolume).Should(Equal(40))
			Ω(c.DeltaTSmoke).Should(Equal(20))
			Ω(c.DeltaTFan).Should(Equal(30))
			Ω(c.DeltaTPump).Should(Equal(60))
			Ω(c.HRMMacAddress).Should(Equal("FF:FF:FF:FF:FF:FF"))
			Ω(c.GPIOPinFan).Should(Equal(1))
			Ω(c.GPIOPinPump).Should(Equal(2))
			Ω(c.BeatRate).Should(BeNumerically("~", 0.8, 0.001))
			Ω(c.S1Beat.Red).Should(Equal(100))
		})
	})
})
