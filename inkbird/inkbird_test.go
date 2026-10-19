package inkbird_test

import (
	"testing"

	"github.com/avanha/pmaas-bluetooth-devices/inkbird"
	"github.com/avanha/pmaas-spi/bluetooth"
	"github.com/avanha/pmaas-spi/bluetooth/bttest"
)

// Advertisements captured from a Bluetooth proxy, from the sensor that is ours.
const (
	frame      = "0201060302f0ff04097370730aff6607970f0099606408"
	frame2     = "0201060302f0ff04097370730aff8007e30e00d13c6408"
	appleFrame = "02011a020a0c0aff4c001005031cbf6469"
)

var reading = []byte{0x97, 0x0f, 0, 0x99, 0x60, 0x64, 0x08}

func hear(data map[uint16][]byte) bluetooth.Result {
	return inkbird.IBSTH.New().Hear(&bluetooth.AdvertisementEvent{Fields: bluetooth.Fields{ManufacturerData: data}}, bttest.Start)
}

func TestReadsTheCapturedAdvertisements(t *testing.T) {
	// The temperature is the "company ID": 0x0766 is 1894, so 18.94 degrees; 0x0f97 is 3991, so 39.91 percent.
	results := bttest.Feed(inkbird.IBSTH.New(), bttest.Advert(t, "49:22:01:23:0A:1B", frame), bttest.Advert(t, "49:22:01:23:0A:1B", frame2))

	if r := results[0]; r.Reading.Temperature != 18.94 || r.Reading.Humidity != 39.91 || *r.Battery != 100 {
		t.Errorf("%+v", r.Reading)
	}

	if r := results[1]; r.Reading.Temperature != 19.20 || r.Reading.Humidity != 38.11 || *r.Battery != 100 {
		t.Errorf("%+v", r.Reading)
	}
}

func TestPositiveZeroAndNegativeTemperatures(t *testing.T) {
	data := []byte{237, 18, 0, 153, 47, 74, 8} // humidity 48.45, battery 74

	for name, c := range map[string]struct {
		tag  uint16
		want float32
	}{
		"positive": {uint16(int(5.36 * 100)), 5.36},
		"zero":     {0, 0},
		"negative": {0xff06, -2.5}, // -250 as an unsigned 16 bit number
	} {
		result := hear(map[uint16][]byte{c.tag: data})
		if result.Reading == nil || result.Reading.Temperature != c.want || result.Reading.Humidity != 48.45 || *result.Battery != 74 {
			t.Errorf("%s: %+v", name, result)
		}
	}
}

func TestTheSameTemperatureWithADifferentHumidityIsADifferentReading(t *testing.T) {
	tag := uint16(int(5.36 * 100))

	first := hear(map[uint16][]byte{tag: {237, 18, 0, 153, 47, 74, 8}})
	second := hear(map[uint16][]byte{tag: {236, 18, 0, 153, 47, 74, 8}})

	if first.Reading.Humidity != 48.45 || second.Reading.Humidity != 48.44 {
		t.Errorf("humidity %v and %v", first.Reading.Humidity, second.Reading.Humidity)
	}
}

func TestMoreThanOneReadingInOneAdvertisementCantBeToldApartSoNoneIsReported(t *testing.T) {
	if result := hear(map[uint16][]byte{0x0766: reading, 0x0767: reading}); !result.IsEmpty() {
		t.Errorf("%+v", result)
	}
}

func TestOnlyDataOfTheRightLengthIsAReading(t *testing.T) {
	for name, data := range map[string][]byte{"short": {1, 2, 3}, "long": make([]byte, 8), "none": nil} {
		if result := hear(map[uint16][]byte{0x0766: data}); !result.IsEmpty() {
			t.Errorf("%s: %+v", name, result)
		}
	}

	// A reading next to a record of something else (Apple's, say) is still the one reading.
	if result := hear(map[uint16][]byte{0x0766: reading, 0x004c: {2, 21}}); result.Reading == nil {
		t.Error("a reading beside another company's record was not read")
	}
}

func TestIdentifiesOnlyByNameAndServiceTogether(t *testing.T) {
	if !inkbird.IBSTH.Identify(bttest.Known(t, frame)) {
		t.Error("not identified")
	}

	if inkbird.IBSTH.Identify(bttest.Known(t, appleFrame)) {
		t.Error("an Apple device was identified")
	}

	// The service alone: another device lists it too.
	known := bttest.Known(t, frame)
	known.LocalName = ""

	if inkbird.IBSTH.Identify(known) {
		t.Error("identified without its name")
	}

	known.LocalName = "tps"
	if inkbird.IBSTH.Identify(known) {
		t.Error("identified by another name")
	}

	known = bttest.Known(t, frame)
	known.ServiceUUIDs = nil

	if inkbird.IBSTH.Identify(known) {
		t.Error("identified without its service")
	}
}

func TestRulesAreValidAndWantTheirOwnDevicesAndNothingElse(t *testing.T) {
	for _, rule := range inkbird.IBSTH.Rules() {
		if err := rule.Validate(); err != nil {
			t.Errorf("%s: %v", rule.Name, err)
		}
	}

	wanted := func(data string) bool {
		return bluetooth.MatchAny(inkbird.IBSTH.Rules(), bttest.Advert(t, "49:22:01:23:0A:1B", data)) != nil
	}

	if !wanted(frame) || wanted(appleFrame) {
		t.Error("the rules want the wrong things")
	}
}

func TestItNeedsTheScanResponseAndTheConfiguredOneSaysSo(t *testing.T) {
	c := inkbird.Thermometer("49:22:01:23:0a:1b", "Patio")

	if !inkbird.IBSTH.NeedsScanResponse() || !c.NeedsScanResponse() || c.(bluetooth.MakeAndModeler).MakeAndModel() != "Inkbird IBS-TH" {
		t.Errorf("%+v", c)
	}

	if !c.Claims(bttest.Advert(t, "49:22:01:23:0A:1B", frame)) || c.Claims(bttest.Advert(t, "49:22:01:23:0A:1C", frame)) {
		t.Error("claims the wrong advertisements")
	}
}
