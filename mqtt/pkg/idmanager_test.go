package pkg

import (
	"math"
	"testing"

	"github.com/tada/mqtt-nats/jsonstream"

	"github.com/tada/mqtt-nats/testutils"
)

func TestIdManager_NextFreePacketID_flip(t *testing.T) {
	idm := NewIDManager().(*idManager)
	idm.nextFreePkgID = math.MaxUint16 - 1
	testutils.CheckEqual(uint16(math.MaxUint16), idm.NextFreePacketID(), t)
	testutils.CheckEqual(uint16(1), idm.NextFreePacketID(), t)
}

func TestIdManager_NextFreePacketID_flipInFlight(t *testing.T) {
	idm := NewIDManager().(*idManager)
	idm.nextFreePkgID = math.MaxUint16 - 2
	idm.inFlight[uint16(math.MaxUint16)] = true
	testutils.CheckEqual(uint16(math.MaxUint16-1), idm.NextFreePacketID(), t)
	testutils.CheckEqual(uint16(1), idm.NextFreePacketID(), t)
}

func TestIdManager_NextFreePacketID_json(t *testing.T) {
	idm := NewIDManager().(*idManager)
	idm.nextFreePkgID = math.MaxUint16 - 3
	idm.inFlight[uint16(math.MaxUint16-1)] = true
	idm.inFlight[uint16(math.MaxUint16)] = true
	idm.inFlight[uint16(1)] = true
	bs, err := jsonstream.Marshal(idm)
	testutils.CheckNotError(err, t)
	idm = &idManager{}
	testutils.CheckNotError(jsonstream.Unmarshal(idm, bs), t)
	testutils.CheckEqual(uint16(math.MaxUint16-2), idm.NextFreePacketID(), t)
	testutils.CheckEqual(uint16(2), idm.NextFreePacketID(), t)
}
