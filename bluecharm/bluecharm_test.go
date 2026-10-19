package bluecharm_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/avanha/pmaas-bluetooth-devices/bluecharm"
	"github.com/avanha/pmaas-spi/bluetooth"
	"github.com/avanha/pmaas-spi/bluetooth/bttest"
	"github.com/avanha/pmaas-spi/device"
)

const address = "DD:88:00:00:3D:36"

// The UUIDs a BlueCharm BC011 Pro advertises after a single and a double click, as captured.
const (
	singleUUID = "426c7565-4368-6172-6d42-6561636f6e77"
	doubleUUID = "7777772e-6b6b-6d63-6e2e-636f6d000006"
)

func ibeaconFrame(uuid string, major uint16, minor uint16) string {
	compact := ""
	for _, c := range uuid {
		if c != '-' {
			compact += string(c)
		}
	}

	return fmt.Sprintf("0201061aff4c000215%s%04x%04xc9", compact, major, minor)
}

// tlmFrame is the advertising data of an Eddystone-TLM frame with the battery at the given voltage.
func tlmFrame(millivolts uint16) string {
	return fmt.Sprintf("0201060303aafe1116aafe2000%04x1880000001000000ffff", millivolts)
}

// systemFrame is the System frame as a proxy captured it, with the battery at the given percentage, and its name in the scan response.
func systemFrame(percent byte) string {
	return fmt.Sprintf("0201060303aafe0e16aafe220b%02xdd8800003d36063e0d09424350726f5f323232363136", percent)
}

func newButton() *bluecharm.BeaconButton {
	return bluecharm.Button("dd-88-00-00-3d-36", "Couch").
		Single(bluetooth.MatchIBeacon(singleUUID)).
		Double(bluetooth.MatchIBeacon(doubleUUID))
}

func TestSystemFrameParses(t *testing.T) {
	got, ok := bluecharm.ParseSystem(bttest.Advert(t, address, systemFrame(0x67)).ServiceData)
	if !ok {
		t.Fatal("not recognised")
	}

	// 0x0b is the model, 0x67 the battery (103%: over 100 on a good cell), then the address, and the firmware, 6.62.
	if got.Model != 11 || got.BatteryPercent != 103 || got.Address != address || got.FirmwareMajor != 6 || got.FirmwareMinor != 62 {
		t.Errorf("got %+v", got)
	}
}

func TestSystemFrameRefusesWhatIsNotOne(t *testing.T) {
	const feaa = "0000feaa-0000-1000-8000-00805f9b34fb"

	frame := []byte{0x22, 0x0b, 0x67, 0xdd, 0x88, 0x00, 0x00, 0x3d, 0x36, 0x06, 0x3e}

	for name, data := range map[string]map[string][]byte{
		"nothing":      nil,
		"another UUID": {"0000feab-0000-1000-8000-00805f9b34fb": frame},
		"a TLM frame":  {feaa: append([]byte{0x20}, frame[1:]...)},
		"short":        {feaa: frame[:10]},
	} {
		if _, ok := bluecharm.ParseSystem(data); ok {
			t.Errorf("%s was accepted", name)
		}
	}
}

func TestAPressIsAnnouncedByTheFirstFrameOfItAndNotTheRestOfTheBurst(t *testing.T) {
	d := newButton().New()
	double := func() *bluetooth.AdvertisementEvent { return bttest.Advert(t, address, ibeaconFrame(doubleUUID, 1, 1)) }

	results := bttest.FeedEvery(d, time.Second, double(), double(), double())
	if len(results[0].Presses) != 1 || results[0].Presses[0] != device.PressDouble || len(results[1].Presses) != 0 || len(results[2].Presses) != 0 {
		t.Errorf("%+v", results)
	}

	// Long after the last frame (the clock starts again, so the first is not): another press.
	results = bttest.FeedEvery(d, 8*time.Second, double(), double())
	if len(results[0].Presses) != 0 || len(results[1].Presses) != 1 {
		t.Errorf("%+v", results)
	}
}

func TestBatteryFromTheSystemFrameIsTheBeaconsOwnFigureAndAVoltageDoesNotReplaceIt(t *testing.T) {
	d := newButton().New()

	results := bttest.Feed(d,
		bttest.Advert(t, address, tlmFrame(2800)), // 55% by the curve
		bttest.Advert(t, address, systemFrame(0x67)),
		bttest.Advert(t, address, systemFrame(62)),
		bttest.Advert(t, address, tlmFrame(2700)))

	if *results[0].Battery != 55 || *results[0].Millivolts != 2800 {
		t.Errorf("from the voltage: %+v", results[0])
	}

	// 103%: over 100 on a good cell.
	if *results[1].Battery != 100 || results[1].Millivolts != nil || *results[2].Battery != 62 {
		t.Errorf("from the System frame: %+v %+v", results[1], results[2])
	}

	if results[3].Battery != nil || *results[3].Millivolts != 2700 {
		t.Errorf("after the beacon gave its own level: %+v", results[3])
	}

	for i, r := range results {
		if len(r.Presses) != 0 {
			t.Errorf("frame %d was taken for a press", i)
		}
	}
}

func TestBatteryLevelFallsAwayAsTheVoltageDoes(t *testing.T) {
	last := 101

	for _, mv := range []uint16{3300, 3000, 2950, 2900, 2850, 2800, 2700, 2650, 2600, 2500, 2400, 2200, 2000, 1500} {
		r := newButton().New().Hear(bttest.Advert(t, address, tlmFrame(mv)), bttest.Start)
		if got := *r.Battery; got < 0 || got > 100 || got > last {
			t.Errorf("%d mV is %d%% after %d%%", mv, got, last)
		} else {
			last = got
		}
	}

	for mv, want := range map[uint16]int{3300: 100, 3000: 100, 2900: 80, 2800: 55, 2400: 5, 2000: 0, 1000: 0, 2950: 90} {
		if got := *newButton().New().Hear(bttest.Advert(t, address, tlmFrame(mv)), bttest.Start).Battery; got != want {
			t.Errorf("%d mV is %d%%, want %d%%", mv, got, want)
		}
	}
}

func TestConfiguration(t *testing.T) {
	b := newButton()

	if b.Address() != address || b.Name() != "Couch" || b.Kind() != bluetooth.KindButton || b.Validate() != nil || b.NeedsScanResponse() {
		t.Errorf("%+v", b)
	}

	rules := b.Rules()
	if len(rules) != 1 || rules[0].Latency != bluetooth.Immediate || rules[0].Address != address || rules[0].Validate() != nil {
		t.Errorf("%+v", rules)
	}

	if !b.Claims(bttest.Advert(t, address, "020106")) || b.Claims(bttest.Advert(t, "DD:88:00:00:3D:37", "020106")) {
		t.Error("claims the wrong advertisements")
	}

	// Before the plugin has it, it is a button never pressed.
	if data := b.GetButtonData(); data.Name != "Couch" || data.Presses != 0 || b.EntityId() != "" {
		t.Errorf("%+v", data)
	}

	if bluecharm.Button(address, "x").Validate() == nil {
		t.Error("a button with no presses was valid")
	}

	defer func() {
		if recover() == nil {
			t.Error("a bad address did not panic")
		}
	}()

	bluecharm.Button("nope", "x")
}
