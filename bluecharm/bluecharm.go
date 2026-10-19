// Package bluecharm knows BlueCharm's beacons (made by KKM), such as the BC011 Pro, used as buttons: with its triggers set, it
// advertises a frame for a few seconds after it is pressed, a different one for each way of pressing it. It also advertises a
// System frame or an Eddystone-TLM frame, in a slot set to either, which is where the battery comes from.
package bluecharm

import (
	"fmt"
	"time"

	"github.com/avanha/pmaas-spi/bluetooth"
)

// All is every model in the package. There are none: a button is configured (see Button), because which frame is which press is
// the owner's choice.
func All() []bluetooth.DeviceModel {
	return nil
}

// BeaconButton is a BlueCharm beacon used as a button, which the assembly says the frames of. It is also the handle to the button
// entity the bluetooth plugin makes for it (EntityId, GetButtonData), so it goes wherever a device.IButton does. Before the plugin
// has started, and after it stops, it answers as a button that has not been pressed.
type BeaconButton struct {
	*bluetooth.ButtonHandle
	address string
	name    string
	presses *bluetooth.IBeaconPresses
}

var _ bluetooth.ConfiguredDevice = (*BeaconButton)(nil)
var _ bluetooth.Addressed = (*BeaconButton)(nil)
var _ bluetooth.MakeAndModeler = (*BeaconButton)(nil)
var _ bluetooth.Validator = (*BeaconButton)(nil)
var _ bluetooth.ButtonBinder = (*BeaconButton)(nil)

// Button is the beacon at the address (any case, ":" or "-"). Say which frame is which press on what it returns; at least one
// is needed. It panics if the address is not a Bluetooth address, like regexp.MustCompile does for a pattern that can't be: it is
// written in the program.
func Button(address string, name string) *BeaconButton {
	normal, ok := bluetooth.NormalizeAddress(address)
	if !ok {
		panic(fmt.Sprintf("Button address %q is not a Bluetooth address like AA:BB:CC:DD:EE:FF", address))
	}

	return &BeaconButton{ButtonHandle: bluetooth.NewButtonHandle(name), address: normal, name: name, presses: bluetooth.NewIBeaconPresses(name)}
}

// Single is the frame the beacon advertises after a single click. Each press needs a frame of its own. It panics if another press
// has the same frame.
func (b *BeaconButton) Single(frame bluetooth.IBeaconMatch) *BeaconButton {
	b.presses.Single(frame)
	return b
}

// Double is the frame the beacon advertises after a double click.
func (b *BeaconButton) Double(frame bluetooth.IBeaconMatch) *BeaconButton {
	b.presses.Double(frame)
	return b
}

// Triple is the frame the beacon advertises after a triple click.
func (b *BeaconButton) Triple(frame bluetooth.IBeaconMatch) *BeaconButton {
	b.presses.Triple(frame)
	return b
}

// Long is the frame the beacon advertises after a long press.
func (b *BeaconButton) Long(frame bluetooth.IBeaconMatch) *BeaconButton {
	b.presses.Long(frame)
	return b
}

// Burst is how long the beacon goes on advertising after a press (its "Trigger Adv Duration"). The default is three seconds. See
// bluetooth.IBeaconPresses.Burst.
func (b *BeaconButton) Burst(burst time.Duration) *BeaconButton { b.presses.Burst(burst); return b }

func (b *BeaconButton) Name() string                                { return b.name }
func (b *BeaconButton) Kind() bluetooth.Kind                        { return bluetooth.KindButton }
func (b *BeaconButton) Address() string                             { return b.address }
func (b *BeaconButton) MakeAndModel() string                        { return "BlueCharm BC011 Pro" }
func (b *BeaconButton) NeedsScanResponse() bool                     { return false }
func (b *BeaconButton) Validate() error                             { return b.presses.Validate() }
func (b *BeaconButton) Claims(a *bluetooth.AdvertisementEvent) bool { return a.Address == b.address }

// Rules want every frame as it is heard: the first of a burst is the press, and nothing may wait.
func (b *BeaconButton) Rules() []bluetooth.Rule {
	return []bluetooth.Rule{{Name: "button " + b.name, Address: b.address, Latency: bluetooth.Immediate}}
}

func (b *BeaconButton) New() bluetooth.Device {
	return &buttonDevice{hearer: b.presses.NewHearer()}
}

type buttonDevice struct {
	hearer *bluetooth.PressHearer

	// ownLevel is that the battery level is the beacon's own (a System frame), so a voltage doesn't replace it.
	ownLevel bool
}

func (d *buttonDevice) Hear(a *bluetooth.AdvertisementEvent, now time.Time) bluetooth.Result {
	var result bluetooth.Result

	// The battery, from whichever frame the beacon is set to advertise. Its own figure (the System frame) is better than one worked
	// out from a voltage (Eddystone-TLM), so once the beacon has given its own, the voltage only adds itself to the picture.
	if system, ok := ParseSystem(a.ServiceData); ok {
		result.Battery = bluetooth.Percent(min(int(system.BatteryPercent), 100))
		d.ownLevel = true
	} else if tlm, ok := bluetooth.ParseEddystoneTLM(a.ServiceData); ok && tlm.BatteryMillivolts > 0 {
		result.Millivolts = bluetooth.Percent(int(tlm.BatteryMillivolts))

		if !d.ownLevel {
			result.Battery = bluetooth.Percent(batteryLevel(int(tlm.BatteryMillivolts)))
		}
	}

	result.Presses = d.hearer.Hear(a, now)

	return result
}
