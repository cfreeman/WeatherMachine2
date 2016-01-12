# WeatherMachine2

This is the control software for [Nathen Street's](http://www.nathenstreet.com/) second iteration of his weather machine installation.


## Compilation / Installation (OSX)

```
	$ mkdir weather-machine-ii
	$ cd weather-machine-ii
	$ export GOPATH=`pwd`
	$ go get github.com/onsi/ginkgo
	$ go get github.com/onsi/gomega
	$ go get github.com/kidoman/embd

```

## Running notes
```
	$ sudo su
	$ hciconfig hci0 down
	$ service bluetooth stop
	$ ./WeatherMachine2
```

## TODO:
* ~~Setup and installation on Raspbian.~~
* Installation details.
* Tidy up a few odds and ends.
* ~~Add delay between pulse and starting smoke.~~
* ~~Add in new state, light always on when we pair with the HRM. TO be finalised~~
* ~~Smoke machine on duration, milliseconds.~~
* ~~Stub out main loop for processing installation logic.~~
	* ~~Looks to see if we need to clean up the ticker.~~
	* ~~Mechanics for polling HRM and pushing that into logic for the installation.~~
	* ~~Tidy up hrm.go Remove testing Main.~~
* ~~Connect to BLE HRM.~~
* ~~Implement the BLE HRP protocol for detecting HR.~~
* ~~Connect to DMX controller for outputting DMX values.~~
* ~~Craft DMX messages for controlling smoke machine.~~
* ~~Pi2 GPIO port control.~~



## License

Copyright (c) 2015 Clinton Freeman

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.