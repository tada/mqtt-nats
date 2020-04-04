package pkg

import "github.com/tada/mqtt-nats/mqtt"

// The PingRequest type represents the MQTT PINGREQ package
type PingRequest int

// PingRequestSingleton is the one and only instance of the PingRequest type
const PingRequestSingleton = PingRequest(0)

// ID always returns 0 for a PINGREQ package
func (a PingRequest) ID() uint16 {
	return 0
}

// Equals returns true if this package is equal to the given package, false if not
func (PingRequest) Equals(p Package) bool {
	return p == PingRequestSingleton
}

// String returns a brief string representation of the package. Suitable for logging
func (PingRequest) String() string {
	return "PINGREQ"
}

// Write writes the MQTT bits of this package on the given Writer
func (PingRequest) Write(w *mqtt.Writer) {
	w.WriteU8(TpPing)
	w.WriteU8(0)
}

// The PingResponse type represents the MQTT PINGRESP package
type PingResponse int

// PingResponseSingleton is the one and only instance of the PingResponse type
const PingResponseSingleton = PingResponse(0)

// ID always returns 0 for a PINGRESP package
func (a PingResponse) ID() uint16 {
	return 0
}

// Equals returns true if this package is equal to the given package, false if not
func (PingResponse) Equals(p Package) bool {
	return p == PingResponseSingleton
}

// String returns a brief string representation of the package. Suitable for logging
func (PingResponse) String() string {
	return "PINGRESP"
}

// Write writes the MQTT bits of this package on the given Writer
func (PingResponse) Write(w *mqtt.Writer) {
	w.WriteU8(TpPingResp)
	w.WriteU8(0)
}
