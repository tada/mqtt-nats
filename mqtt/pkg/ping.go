package pkg

import "github.com/tada/mqtt-nats/mqtt"

type PingRequest int

const PingRequestSingleton = PingRequest(0)
const PingResponseSingleton = PingResponse(0)

func (PingRequest) Equals(p Package) bool {
	return p == PingRequestSingleton
}

func (PingRequest) String() string {
	return "PINGREQ"
}

func (PingRequest) Type() byte {
	return TpPing
}

func (PingRequest) Write(w *mqtt.Writer) {
	w.WriteU8(TpPing)
	w.WriteU8(0)
}

type PingResponse int

func (PingResponse) Equals(p Package) bool {
	return p == PingResponseSingleton
}

func (PingResponse) String() string {
	return "PINGRESP"
}

func (PingResponse) Type() byte {
	return TpPingResp
}

func (PingResponse) Write(w *mqtt.Writer) {
	w.WriteU8(TpPingResp)
	w.WriteU8(0)
}
