package mqtt

import (
	"testing"
)

func TestFromNATS(t *testing.T) {
	tests := []struct {
		name string
		nats string
		want string
	}{{
		name: "Slash is dot",
		nats: "a/b/c",
		want: "a.b.c",
	}, {
		name: "Dot is slash",
		nats: "a.b.c",
		want: "a/b/c",
	},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := FromNATS(tt.nats); got != tt.want {
				t.Errorf("FromNATS() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFromNATSSubscription(t *testing.T) {
	tests := []struct {
		name string
		nats string
		want string
	}{{
		name: "Slash is dot",
		nats: "a/b/c",
		want: "a.b.c",
	}, {
		name: "Dot is slash",
		nats: "a.b.c",
		want: "a/b/c",
	}, {
		name: "Star is plus",
		nats: "a.*.b",
		want: "a/+/b",
	}, {
		name: "Plus is star",
		nats: "a.+.b",
		want: "a/*/b",
	}, {
		name: "> is #",
		nats: "a.b.>",
		want: "a/b/#",
	}, {
		name: "# is >",
		nats: "a.b.#",
		want: "a/b/>",
	}}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := FromNATSSubscription(tt.nats); got != tt.want {
				t.Errorf("FromNATSSubscription() = %v, want %v", got, tt.want)
			}
		})
	}
}
