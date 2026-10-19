package bluecharm

import "fmt"

const (
	eddystoneUUID = "0000feaa-0000-1000-8000-00805f9b34fb"

	systemType      = 0x22
	systemMinLength = 11
)

// System is what a KKM "System" frame says. KKM makes the beacons sold as BlueCharm (the BC011 Pro among them), and the frame is
// theirs rather than a standard: it carries who the beacon is and what its battery is at, and is advertised in the same service
// data as Eddystone's frames (0xFEAA). Its layout is in KKM's open source SDK (KBAdvPacketSystem): a frame type of 0x22, the
// model, the battery as a percentage, the beacon's MAC address and the firmware version, major and minor.
type System struct {
	Model uint8

	// BatteryPercent is the beacon's own figure for its battery, which can be over 100 on a new cell.
	BatteryPercent uint8

	// Address is the beacon's address, "DD:88:00:00:3D:36".
	Address string

	FirmwareMajor, FirmwareMinor uint8
}

// ParseSystem finds the System frame in an advertisement's service data, if there is one.
func ParseSystem(serviceData map[string][]byte) (System, bool) {
	data, ok := serviceData[eddystoneUUID]
	if !ok || len(data) < systemMinLength || data[0] != systemType {
		return System{}, false
	}

	return System{
		Model:          data[1],
		BatteryPercent: data[2],
		Address:        fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X", data[3], data[4], data[5], data[6], data[7], data[8]),
		FirmwareMajor:  data[9],
		FirmwareMinor:  data[10],
	}, true
}

// batteryLevel is a rough battery level from a coin cell's voltage: they hold close to their rated voltage for most of their life
// and then fall away, which is why the steps are not even. It is what the beacon's owner wants to know (is it time to change
// it), not a measurement.
func batteryLevel(millivolts int) int {
	steps := [...]struct{ millivolts, percent int }{{3000, 100}, {2900, 80}, {2800, 55}, {2700, 30}, {2600, 15}, {2400, 5}, {2000, 0}}

	if millivolts >= steps[0].millivolts {
		return 100
	}

	for i := 1; i < len(steps); i++ {
		hi, lo := steps[i-1], steps[i]
		if millivolts >= lo.millivolts {
			return lo.percent + (hi.percent-lo.percent)*(millivolts-lo.millivolts)/(hi.millivolts-lo.millivolts)
		}
	}

	return 0
}
