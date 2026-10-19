// Package inkbird knows Inkbird's thermometers.
package inkbird

import (
	"encoding/binary"
	"time"

	"github.com/avanha/pmaas-spi/bluetooth"
)

// serviceUUID is the service the IBS-TH lists.
const serviceUUID = "0000fff0-0000-1000-8000-00805f9b34fb"

// advertisedName is the name the IBS-TH advertises.
const advertisedName = "sps"

// readingLength is how long the data under the "company ID" is.
const readingLength = 7

// IBSTH is the Inkbird IBS-TH thermometer and hygrometer. It puts its reading where a company ID goes: the 16-bit signed
// temperature in hundredths of a degree is the "company ID", and the humidity, probe, modbus and battery follow. So every
// reading is a different key.
//
// Its name, "sps", is only in its scan response, and so is its reading, so a passive scan never sees it; and the service it
// lists is used by other gadgets. It is safest configured by address (see Thermometer), which also makes the bluetooth plugin scan
// actively now and then to hear it.
var IBSTH bluetooth.DeviceModel = ibsth{}

// All is every model in the package.
func All() []bluetooth.DeviceModel {
	return []bluetooth.DeviceModel{IBSTH}
}

// Thermometer is the Inkbird IBS-TH at the address, which you say is one. It panics if the address is not one.
func Thermometer(address string, name string) bluetooth.ConfiguredDevice {
	return bluetooth.ConfigureModel(IBSTH, address, name)
}

type ibsth struct{}

func (ibsth) Name() string            { return "Inkbird IBS-TH" }
func (ibsth) Kind() bluetooth.Kind    { return bluetooth.KindThermometer }
func (ibsth) NeedsScanResponse() bool { return true }
func (ibsth) New() bluetooth.Device   { return ibsth{} }

func (ibsth) Rules() []bluetooth.Rule {
	return []bluetooth.Rule{{Name: "inkbird-ibs-th", ServiceUUID: serviceUUID, NamePrefix: advertisedName}}
}

func (ibsth) Identify(known bluetooth.Fields) bool {
	if known.LocalName != advertisedName {
		return false
	}

	for _, uuid := range known.ServiceUUIDs {
		if uuid == serviceUUID {
			return true
		}
	}

	return false
}

// Hear decodes the reading, if there is exactly one in the advertisement. With more than one it can't tell which is the latest,
// and a guess risks reporting a stale reading, so it waits for the next.
func (ibsth) Hear(a *bluetooth.AdvertisementEvent, _ time.Time) bluetooth.Result {
	var tag uint16
	var data []byte
	var found int

	for key, bytes := range a.ManufacturerData {
		if len(bytes) == readingLength {
			tag, data = key, bytes
			found++
		}
	}

	if found != 1 {
		return bluetooth.Result{}
	}

	// (temp, hum, probe, modbus, bat) = unpack("<hHBHB", ...). The temperature is the "company ID", unsigned as we read it.
	temperature := int16(tag)
	humidity := binary.LittleEndian.Uint16(data[0:2])

	return bluetooth.Result{
		Battery: bluetooth.Percent(int(data[5])),
		Reading: &bluetooth.Environment{
			Temperature: float32(temperature) / float32(100),
			Humidity:    float32(humidity) / float32(100),
			HasHumidity: true,
		},
	}
}
