package types

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"strings"
	"time"
)

type IsoTime time.Time

// Common layouts you'll encounter (add more if needed)
var isoLayouts = []string{
	time.RFC3339Nano, // 2006-01-02T15:04:05.999999999Z07:00
	time.RFC3339,     // 2006-01-02T15:04:05Z07:00
	"2006-01-02T15:04:05.000Z07:00",
	"2006-01-02T15:04:05.000Z", // your example with ".000Z"
	"2006-01-02T15:04:05Z",     // no fractional seconds, UTC Z
}

func parseIso(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, layout := range isoLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	// last resort: try RFC3339 again after normalizing Z
	if strings.HasSuffix(s, "Z") && !strings.HasSuffix(s, "+00:00") {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return t, nil
		}
	}
	_, err := time.Parse(time.RFC3339Nano, s)
	return time.Time{}, err // return a parse error
}

// For printing nicely when you fmt.Println(IsoTime)
func (t IsoTime) String() string {
	return time.Time(t).UTC().Format("2006-01-02T15:04:05.000Z")
}

func (t IsoTime) Time() time.Time { return time.Time(t) }

// ----- XML -----

func (t IsoTime) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	v := time.Time(t).UTC().Format("2006-01-02T15:04:05.000Z")
	return e.EncodeElement(v, start)
}

func (t *IsoTime) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var s string
	if err := d.DecodeElement(&s, &start); err != nil {
		return err
	}
	if s == "" {
		*t = IsoTime(time.Time{}) // zero value
		return nil
	}
	tt, err := parseIso(s)
	if err != nil {
		return err
	}
	*t = IsoTime(tt)
	return nil
}

// ----- JSON -----

func (t IsoTime) MarshalJSON() ([]byte, error) {
	s := time.Time(t).UTC().Format("2006-01-02T15:04:05.000Z")
	return json.Marshal(s)
}

func (t *IsoTime) UnmarshalJSON(b []byte) error {
	// handle null
	if bytes.Equal(b, []byte("null")) {
		*t = IsoTime(time.Time{})
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s == "" {
		*t = IsoTime(time.Time{})
		return nil
	}
	tt, err := parseIso(s)
	if err != nil {
		return err
	}
	*t = IsoTime(tt)
	return nil
}
