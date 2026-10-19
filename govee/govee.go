// Package govee knows Govee's thermometers.
package govee

import (
	"encoding/binary"
	"strings"
	"time"

	"github.com/avanha/pmaas-spi/bluetooth"
)

// companyID is the company ID Govee's H5051/H5071/H5072/H5074/H5075 put their readings under, and also the 16-bit service UUID
// they list.
const companyID = 0xEC88

// uuidPrefix is what a Govee's listed service UUID begins with.
const uuidPrefix = "0000ec88"

// H5075 is the Govee GVH5075 thermometer and hygrometer. Its reading is in the manufacturer data under company ID 0xEC88: a
// 24-bit number from which the temperature and humidity are worked out, and then the battery percentage.
var H5075 bluetooth.DeviceModel = h5075{}

// All is every model in the package.
func All() []bluetooth.DeviceModel {
	return []bluetooth.DeviceModel{H5075}
}

// Thermometer names the GVH5075 at the address. It is recognised without this: it is for a name. It panics if the address is not
// one.
func Thermometer(address string, name string) bluetooth.ConfiguredDevice {
	return bluetooth.ConfigureModel(H5075, address, name)
}

type h5075 struct{}

func (h5075) Name() string            { return "Govee GVH5075" }
func (h5075) Kind() bluetooth.Kind    { return bluetooth.KindThermometer }
func (h5075) NeedsScanResponse() bool { return false }
func (h5075) New() bluetooth.Device   { return h5075{} }

func (h5075) Rules() []bluetooth.Rule {
	return []bluetooth.Rule{{Name: "govee-h5075", ServiceUUID: bluetooth.UUID16(companyID)}}
}

func (h5075) Identify(known bluetooth.Fields) bool {
	for _, uuid := range known.ServiceUUIDs {
		if strings.HasPrefix(uuid, uuidPrefix) {
			return true
		}
	}

	return false
}

func (h5075) Hear(a *bluetooth.AdvertisementEvent, _ time.Time) bluetooth.Result {
	data, ok := a.ManufacturerData[companyID]

	// Four bytes of reading and one of battery. An advertisement from something else with this ID can be shorter.
	if !ok || len(data) < 5 {
		return bluetooth.Result{}
	}

	value := binary.BigEndian.Uint32(data[0:4])

	// The top bit of the 24 says the temperature is below zero, and is not part of the number.
	below := value&0x800000 != 0
	reading := value &^ 0x800000

	return bluetooth.Result{
		Battery: bluetooth.Percent(int(data[4])),
		Reading: &bluetooth.Environment{
			Temperature: temperature(reading, below),
			Humidity:    float32(reading%1000) / float32(10),
			HasHumidity: true,
		},
	}
}

// temperature is the temperature in Celsius from the reading, which is the temperature in tenths of a degree followed by three
// digits of humidity.
func temperature(reading uint32, below bool) float32 {
	temp := int32(reading) / 1000

	if below {
		temp = -temp
	}

	return float32(temp) / float32(10)
}
