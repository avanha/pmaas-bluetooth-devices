package govee_test

import (
	"testing"

	"github.com/avanha/pmaas-bluetooth-devices/govee"
	"github.com/avanha/pmaas-spi/bluetooth"
	"github.com/avanha/pmaas-spi/bluetooth/bttest"
)

// Advertisements captured from a Bluetooth proxy, from sensors that are ours.
const (
	frame      = "0d09475648353037355f45324630030388ec02010509ff88ec000339a664001aff4c000215494e54454c4c495f524f434b535f48575075f2ff0c"
	appleFrame = "02011a020a0c0aff4c001005031cbf6469"
	uuidAlone  = "0303" + "88ec" + "09ff88ec000339a66400"
)

func hear(data map[uint16][]byte) bluetooth.Result {
	return govee.H5075.New().Hear(&bluetooth.AdvertisementEvent{Fields: bluetooth.Fields{ManufacturerData: data}}, bttest.Start)
}

func TestReadsTheCapturedAdvertisement(t *testing.T) {
	result := bttest.Feed(govee.H5075.New(), bttest.Advert(t, "A4:C1:38:CC:E2:F0", frame))[0]

	// 0x000339a6 is 211366: 21.1 degrees, 36.6 percent. Battery 0x64.
	if result.Reading == nil || result.Reading.Temperature != 21.1 || result.Reading.Humidity != 36.6 || !result.Reading.HasHumidity ||
		result.Battery == nil || *result.Battery != 100 {
		t.Errorf("%+v", result)
	}
}

func TestBelowZeroTheTopBitSaysSoAndIsNotPartOfTheNumber(t *testing.T) {
	// -0.5 degrees, 12.3 percent: 5123 with the sign bit set, battery 87.
	value := uint32(0x800000 | 5123)

	result := hear(map[uint16][]byte{0xec88: {byte(value >> 24), byte(value >> 16), byte(value >> 8), byte(value), 87}})
	if result.Reading.Temperature != -0.5 || result.Reading.Humidity != 12.3 || *result.Battery != 87 {
		t.Errorf("%+v", result.Reading)
	}

	// -5.1 degrees, 23.4 percent: 51234 with the sign bit set.
	value = 0x800000 | 51234

	result = hear(map[uint16][]byte{0xec88: {byte(value >> 24), byte(value >> 16), byte(value >> 8), byte(value), 50}})
	if result.Reading.Temperature != -5.1 || result.Reading.Humidity != 23.4 {
		t.Errorf("%+v", result.Reading)
	}
}

func TestNothingToReadIsNotAReadingAndShortDataDoesNotPanic(t *testing.T) {
	for name, data := range map[string]map[uint16][]byte{
		"no manufacturer data": nil,
		"another company":      {0x004c: {1, 2, 3, 4, 5, 6}},
		"too short":            {0xec88: {0, 3, 0x39, 0xa6}},
		"empty":                {0xec88: {}},
	} {
		if result := hear(data); !result.IsEmpty() {
			t.Errorf("%s: %+v", name, result)
		}
	}
}

func TestIdentifiesByTheServiceItLists(t *testing.T) {
	if !govee.H5075.Identify(bttest.Known(t, frame)) || !govee.H5075.Identify(bttest.Known(t, uuidAlone)) {
		t.Error("not identified")
	}

	if govee.H5075.Identify(bttest.Known(t, appleFrame)) {
		t.Error("an Apple device was identified")
	}
}

func TestRulesAreValidAndWantTheirOwnDevicesAndNothingElse(t *testing.T) {
	for _, rule := range govee.H5075.Rules() {
		if err := rule.Validate(); err != nil {
			t.Errorf("%s: %v", rule.Name, err)
		}
	}

	wanted := func(data string) bool {
		return bluetooth.MatchAny(govee.H5075.Rules(), bttest.Advert(t, "A4:C1:38:CC:E2:F0", data)) != nil
	}

	if !wanted(frame) || wanted(appleFrame) {
		t.Error("the rules want the wrong things")
	}
}

func TestThermometerIsNamedByAddress(t *testing.T) {
	c := govee.Thermometer("a4-c1-38-cc-e2-f0", "Office")
	if c.(bluetooth.Addressed).Address() != "A4:C1:38:CC:E2:F0" || c.Name() != "Office" || c.Kind() != bluetooth.KindThermometer ||
		c.(bluetooth.MakeAndModeler).MakeAndModel() != "Govee GVH5075" {
		t.Errorf("%+v", c)
	}

	defer func() {
		if recover() == nil {
			t.Error("a bad address did not panic")
		}
	}()

	govee.Thermometer("nope", "x")
}
