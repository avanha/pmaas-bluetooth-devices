# pmaas-bluetooth-devices
Bluetooth device models for the PMAAS bluetooth plugin (`pmaas-plugin-bluetooth`): what a device is, and how to read what it
advertises. A library like `pmaas-automations`, not a plugin: an assembly imports it.

```go
conf := bluetooth.NewPluginConfig()
conf.Recognize(devices.All()...) // every model here: a newer version of this module recognises more devices

// Config is only for what recognition can't give: a name, or a handle for an automation.
conf.AddThermometer("A4:C1:38:CC:E2:F0", "Office")                  // a recognised thermometer, named
conf.Add(inkbird.Thermometer("49:22:01:23:0A:1B", "Patio"))          // one that has to be said (see below)
couch := bluecharm.Button("DD:88:00:00:3D:36", "Couch").            // also a handle: couch is a device.IButton
	Single(bluetooth.MatchAnyIBeacon().WithMajor(0)).
	Double(bluetooth.MatchAnyIBeacon().WithMajor(1))
conf.Add(couch)
```

A builder validates as it is built and panics on a mistake, like `regexp.MustCompile`: the person who wrote the assembly finds out
when it starts.

## Packages

* `govee`: `H5075` (the GVH5075 thermometer and hygrometer). Recognised from the service it lists.
* `inkbird`: `IBSTH`. Its name and reading are only in a scan response, which a passive scan never asks for, and the service it
  lists is used by other gadgets, so it is not recognised on its own: configure it with `inkbird.Thermometer`, which also makes the
  plugin scan actively now and then.
* `bluecharm` (KKM): `Button`, a BC011 Pro with its triggers set. Presses are iBeacon frames, one for each way of pressing; the
  battery is read from a System or an Eddystone-TLM frame.

## Writing a model

A model is in any module; it depends on `pmaas-spi/bluetooth` only. Implement `bluetooth.DeviceModel`:

```go
type DeviceModel interface {
	Name() string               // "Govee GVH5075"
	Kind() Kind                 // KindThermometer, KindButton
	Rules() []Rule              // what the sources should forward, and how fast
	NeedsScanResponse() bool    // its data is in scan responses
	Identify(known Fields) bool // from everything heard of a device so far
	New() Device
}

type Device interface { Hear(a *AdvertisementEvent, now time.Time) Result } // one per device; owns its state
```

* `Identify` sees the fields of everything heard of the device so far merged (a name in one advertisement, a service in another).
  Be strict: a service UUID alone is rarely unique.
* `Rules` are what a source is asked to forward. Make them as narrow as one advertisement allows. Sensors are `Coalesce`; a
  button's frames are `Immediate`.
* `Hear` runs on the plugin's goroutine, one advertisement at a time, so it must be fast and must not block. A panic disables that
  device (and is logged); a slow call is warned about. Return what the advertisement said, and leave the rest nil:
  `Result{Reading: &bluetooth.Environment{...}, Battery: bluetooth.Percent(80)}`.
* A device with a fixed address is configured with `bluetooth.ConfigureModel(model, address, name)`. A button is a
  `ConfiguredDevice` that is `Addressed` and embeds `*bluetooth.ButtonHandle` (see `bluecharm`); `bluetooth.IBeaconPresses` does
  the iBeacon frame to press mapping and burst logic.
* Test with captured advertisements: `bttest.Advert(t, address, "hex of the advertising data")` and `bttest.Feed(device, ads...)`
  (`pmaas-spi/bluetooth/bttest`). Every model here has such tests to copy.

Publish your module and an assembly adds it with one line: `conf.Recognize(acme.All()...)`.
