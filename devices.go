// Package devices is every Bluetooth device model in this module, for the bluetooth plugin to recognise:
//
//	conf := bluetooth.NewPluginConfig()
//	conf.Recognize(devices.All()...)
//
// A newer version of the module knows more devices. Pick from a vendor package (govee.All()) to know fewer, and use its builders
// (inkbird.Thermometer, bluecharm.Button) to name a device or give it to an automation.
package devices

import (
	"github.com/avanha/pmaas-bluetooth-devices/bluecharm"
	"github.com/avanha/pmaas-bluetooth-devices/govee"
	"github.com/avanha/pmaas-bluetooth-devices/inkbird"
	"github.com/avanha/pmaas-spi/bluetooth"
)

// All is every model in the module.
func All() []bluetooth.DeviceModel {
	var models []bluetooth.DeviceModel

	models = append(models, govee.All()...)
	models = append(models, inkbird.All()...)
	models = append(models, bluecharm.All()...)

	return models
}
