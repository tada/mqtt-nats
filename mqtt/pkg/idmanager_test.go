package pkg

import (
	"math"
	"testing"

	"github.com/tada/jsonstream"
	"github.com/tada/mqtt-nats/test/utils"
)

func TestIdManager_NextFreePacketID_flip(t *testing.T) {
	idm := NewIDManager().(*idManager)
	idm.nextFreePkgID = math.MaxUint16 - 1
	utils.CheckEqual(uint16(math.MaxUint16), idm.NextFreePacketID(), t)
	utils.CheckEqual(uint16(1), idm.NextFreePacketID(), t)
}

func TestIdManager_NextFreePacketID_flipInFlight(t *testing.T) {
	idm := NewIDManager().(*idManager)
	idm.nextFreePkgID = math.MaxUint16 - 2
	idm.inFlight[uint16(math.MaxUint16)] = true
	utils.CheckEqual(uint16(math.MaxUint16-1), idm.NextFreePacketID(), t)
	utils.CheckEqual(uint16(1), idm.NextFreePacketID(), t)
}

func TestIdManager_NextFreePacketID_json(t *testing.T) {
	idm := NewIDManager().(*idManager)
	idm.nextFreePkgID = math.MaxUint16 - 3
	idm.inFlight[uint16(math.MaxUint16-1)] = true
	idm.inFlight[uint16(math.MaxUint16)] = true
	idm.inFlight[uint16(1)] = true
	bs, err := jsonstream.Marshal(idm)
	utils.CheckNotError(err, t)
	idm = &idManager{}
	utils.CheckNotError(jsonstream.Unmarshal(idm, bs), t)
	utils.CheckEqual(uint16(math.MaxUint16-2), idm.NextFreePacketID(), t)
	utils.CheckEqual(uint16(2), idm.NextFreePacketID(), t)
}
