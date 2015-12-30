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
	"encoding/binary"
	"log"
	"strings"
	"time"

	"github.com/paypal/gatt"
	"github.com/paypal/gatt/examples/option"
)

// onStateChanged is called when the bluetooth device changes state.
func onStateChanged(d gatt.Device, s gatt.State) {
	switch s {
	case gatt.StatePoweredOn:
		d.Scan([]gatt.UUID{}, false)
		return
	default:
		d.StopScanning()
	}
}

// onPeriphDiscovered is called when a new BLE peripheral is detected by the device.
func onPeriphDiscovered(p gatt.Peripheral, a *gatt.Advertisement, rssi int, deviceID string) {
	if strings.ToUpper(p.ID()) != strings.ToUpper(deviceID) {
		return // This is not the peripheral we're looking for, keep looking.
	}

	// Stop scanning once we've got the peripheral we're looking for.
	p.Device().StopScanning()

	log.Println()
	log.Println("INFO: Peripheral ID =", p.ID())
	log.Println("INFO:  Local Name        =", a.LocalName)
	log.Println("INFO:  TX Power Level    =", a.TxPowerLevel)
	log.Println("INFO:  Manufacturer Data =", a.ManufacturerData)
	log.Println("INFO:  Service Data      =", a.ServiceData)
	log.Println()

	// Connect to the peripheral once we have found it.
	p.Device().Connect(p)
}

// onPeriphConnected is called when we connect to a BLE peripheral.
func onPeriphConnected(p gatt.Peripheral, hr chan int, err error) {
	log.Println("INFO: Connected to BLE Peripheral.")
	defer p.Device().CancelConnection(p)

	if err := p.SetMTU(500); err != nil {
		log.Printf("ERROR: Failed to set MTU - %s\n", err)
	}

	// Get the heart rate service which is identified by the UUID: \x180d
	ss, err := p.DiscoverServices([]gatt.UUID{gatt.MustParseUUID("180d")})
	if err != nil {
		log.Printf("ERROR: Failed to discover services - %s\n", err)
		return
	}

	for _, s := range ss {
		// Get the heart rate measurement characteristic which is identified by the UUID: \x2a37
		cs, err := p.DiscoverCharacteristics([]gatt.UUID{gatt.MustParseUUID("2a37")}, s)
		if err != nil {
			log.Printf("ERROR: Failed to discover characteristics - %s\n", err)
			continue
		}

		for _, c := range cs {
			// Read the characteristic.
			if (c.Properties() & gatt.CharRead) != 0 {
				b, err := p.ReadCharacteristic(c)
				if err != nil {
					log.Printf("ERROR: Failed to read characteristic - %s\n", err)
					continue
				}
				log.Printf("    value         %x | %q\n", b, b)
			}

			// Discover the characteristic descriptors.
			_, err := p.DiscoverDescriptors(nil, c)
			if err != nil {
				log.Printf("Failed to discover descriptors, err: %s\n", err)
				continue
			}

			// Subscribe to any notifications from the characteristic.
			if (c.Properties() & (gatt.CharNotify | gatt.CharIndicate)) != 0 {

				err := p.SetNotifyValue(c, func(c *gatt.Characteristic, b []byte, err error) {
					heartRate := binary.LittleEndian.Uint16(append([]byte(b[1:2]), []byte{0}...))
					hr <- int(heartRate)
					log.Printf("Recieved: % X | %q\n", b, b)
					log.Printf("**HeartRate: %d\n", heartRate)
				})

				if err != nil {
					log.Printf("ERROR: Failed to subscribe characteristic - %s\n", err)
					continue
				}
			}

		}
		log.Println()
	}

	//fmt.Printf("Waiting for 5 seconds to get some notifiations, if any.\n")
	time.Sleep(5 * time.Second)
}

// onPeriphDisconnected is called when a BLE Peripheral is disconnected.
func onPeriphDisconnected(p gatt.Peripheral, err error) {
	log.Println("INFO: Disconnected from BLE peripheral.")
}

// pollHRM connects to the BLE heart rate monitor at deviceID and collects heart rate
// measurements on the channel hr.
func pollHRM(deviceID string, hr chan int) {
	d, err := gatt.NewDevice(option.DefaultClientOptions...)
	if err != nil {
		log.Printf("ERROR: Unable to get bluetooth device.")
		return
	}

	// Register handlers.
	d.Handle(
		gatt.PeripheralDiscovered(func(p gatt.Peripheral, a *gatt.Advertisement, rssi int) {
			onPeriphDiscovered(p, a, rssi, deviceID)
		}),
		gatt.PeripheralConnected(func(p gatt.Peripheral, err error) {
			onPeriphConnected(p, hr, err)
		}),
		gatt.PeripheralDisconnected(onPeriphDisconnected),
	)

	d.Init(onStateChanged)
}
