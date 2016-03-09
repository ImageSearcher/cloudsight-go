package cloudsight

import (
	"fmt"
	"net/url"
)

// Additional API parameters.
type Params map[string]string

func (p Params) values() url.Values {
	values := url.Values{}
	for k, v := range p {
		values[k] = []string{v}
	}
	return values
}

// Set the locale of the request. Default is "en-US".
func (p Params) SetLocale(locale string) {
	p["image_request[locale]"] = locale
}

// Set the language of the request. The response will be returned in this
// language. The default is "en".
func (p Params) SetLanguage(lang string) {
	p["image_request[language]"] = lang
}

// Set a unique ID generated for the device sending the request. We recommend
// generating a UUID.
func (p Params) SetDeviceID(id string) {
	p["image_request[device_id]"] = id
}

// Set the latitude for additional geolocation context.
func (p Params) SetLatitude(lat float64) error {
	if lat > 90.0 || lat < -90.0 {
		return fmt.Errorf("invalid latitude: %f", lat)
	}
	p["image_request[latitude]"] = fmt.Sprint(lat)
	return nil
}

// Set the longitute for additional geolocation context.
func (p Params) SetLongitude(lon float64) error {
	if lon > 180.0 || lon < -180.0 {
		return fmt.Errorf("invalid longitude: %f", lon)
	}
	p["image_request[longitude]"] = fmt.Sprint(lon)
	return nil
}

// Set the altitude for additional geolocation context.
func (p Params) SetAltitude(alt float64) {
	p["image_request[altitude]"] = fmt.Sprint(alt)
}

// Set the position for additional geolocation context.
func (p Params) SetPosition(lat, lon, alt float64) error {
	if err := p.SetLatitude(lat); err != nil {
		return err
	}
	if err := p.SetLongitude(lon); err != nil {
		return err
	}
	p.SetAltitude(alt)
	return nil
}

// Set the deadline in seconds before expired state is set. Use a high ttl for
// low-priority image requests. Use `SetMaxTTL()` for maximum deadline.
func (p Params) SetTTL(ttl int) error {
	if ttl <= 0 {
		return fmt.Errorf("invalid ttl: %d, should be greater than 0", ttl)
	}
	p["image_request[ttl]"] = fmt.Sprint(ttl)
	return nil
}

// Set the maximum deadline before expired state is set.
func (p Params) SetMaxTTL() {
	p["image_request[ttl]"] = "max"
}

// Set a relative focal point on image for specificity.
//
// The point uses North-West gravity (0.0, 0.0 corresponds to upper-left
// corner), for which to place a hightlight of attention on the image. In the
// event there are many identifiable objects in the image, this attempts to
// place importance on the ones closest to the focal point. This method accepts
// relative coordinates (0.0 through 1.0).
func (p Params) SetFocusRelative(x, y float64) error {
	if x < 0.0 || x > 1.0 {
		return fmt.Errorf("invalid focus X parameter: %f, should be [0.0, 1.0]", x)
	}
	if y < 0.0 || y > 1.0 {
		return fmt.Errorf("invalid focus Y parameter: %f, should be [0.0, 1.0]", x)
	}
	p["focus[x]"] = fmt.Sprint(x)
	p["focus[y]"] = fmt.Sprint(y)
	return nil
}

// Set an absolute focal point on image for specificity.
//
// The point uses North-West gravity (0, 0 corresponds to upper-left corner),
// for which to place a hightlight of attention on the image. In the event
// there are many identifiable objects in the image, this attempts to place
// importance on the ones closest to the focal point. This method accepts
// absolute coordinates (ie. a 400x400 image would have 0 through 400 for each
// axis).
func (p Params) SetFocusAbsolute(x, y int) error {
	if x < 0 {
		return fmt.Errorf("invalid focus X parameter: %f, should be greater or equal to 0", x)
	}
	if y < 0 {
		return fmt.Errorf("invalid focus Y parameter: %f, should be greater or equal to 0", x)
	}
	p["focus[x]"] = fmt.Sprint(x)
	p["focus[y]"] = fmt.Sprint(y)
	return nil
}
