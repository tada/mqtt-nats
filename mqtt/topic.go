package mqtt

import (
	"io"
	"regexp"
	"strings"
)

const (
	dot          = rune('.')
	slash        = rune('/')
	star         = rune('*')
	plus         = rune('+')
	hash         = rune('#')
	gt           = rune('>')
	matchSegment = `^[/]+`
	matchRest    = `.*`
)

func SubscriptionToRegexp(s string) *regexp.Regexp {
	w := strings.Builder{}
	for i, p := range strings.Split(s, "/") {
		if i > 0 {
			_ = w.WriteByte('/')
		}
		if len(p) == 1 {
			switch rune(p[0]) {
			case plus:
				_, _ = w.WriteString(matchSegment)
				continue
			case hash:
				_, _ = w.WriteString(matchRest)
				continue
			}
		}
		_, _ = w.WriteString(regexp.QuoteMeta(p))
	}
	return regexp.MustCompile(w.String())
}

// ToNATS converts an MQTT topic to a NATS subject. The following conversions take place
//
// dots become slashes
// slashes become dots
func ToNATS(mqttTopic string) string {
	r := strings.NewReader(mqttTopic)
	w := strings.Builder{}
	for {
		c, _, err := r.ReadRune()
		if err == io.EOF {
			return w.String()
		}
		switch c {
		case dot:
			c = slash
		case slash:
			c = dot
		}
		_, _ = w.WriteRune(c)
	}
}

// ToNATSSubscription converts the given MQTT subscription into a NATS subscription
func ToNATSSubscription(mqttSub string) string {
	r := strings.NewReader(mqttSub)
	w := strings.Builder{}
	for {
		c, _, err := r.ReadRune()
		if err == io.EOF {
			return w.String()
		}
		switch c {
		case dot:
			c = slash
		case slash:
			c = dot
		case star:
			c = plus
		case plus:
			c = star
		case hash:
			c = gt
		case gt:
			c = hash
		}
		_, _ = w.WriteRune(c)
	}
}

// FromNATS converts an MATS subject to a MQTT topic. The following conversions take place
//
// dots become slashes
// slashes become dots
func FromNATS(natsSubject string) string {
	// exact same conversion but opposite direction, at least for now
	return ToNATS(natsSubject)
}

// FromNATSSubscription converts the given NATS subscription into a MQTT subscription
func FromNATSSubscription(natsSubject string) string {
	// exact same conversion but opposite direction, at least for now
	return ToNATSSubscription(natsSubject)
}
