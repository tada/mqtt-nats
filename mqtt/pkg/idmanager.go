package pkg

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/tada/mqtt-nats/jsonstream"
	"github.com/tada/mqtt-nats/pio"
)

// An IDManager manages packet IDs and ensures their uniqueness by maintaining a list of
// IDs that are in use
type IDManager interface {
	// NextFreePacketID allocates and returns the next free packet ID
	NextFreePacketID() uint16

	// ReleasePacketID releases a previously allocated packet ID
	ReleasePacketID(uint16)
}

type idManager struct {
	pkgIDLock     sync.Mutex
	inFlight      map[uint16]bool
	nextFreePkgID uint16
}

// NewIDManager creates a new IDManager
func NewIDManager() IDManager {
	return &idManager{nextFreePkgID: 1, inFlight: make(map[uint16]bool, 37)}
}

func (s *idManager) NextFreePacketID() uint16 {
	s.pkgIDLock.Lock()
	s.nextFreePkgID++
	if s.nextFreePkgID == 0 {
		// counter flipped over and zero is not a valid ID
		s.nextFreePkgID++
	}
	for s.inFlight[s.nextFreePkgID] {
		s.nextFreePkgID++
		if s.nextFreePkgID == 0 {
			// counter flipped over and zero is not a valid ID
			s.nextFreePkgID++
		}
	}
	s.inFlight[s.nextFreePkgID] = true
	s.pkgIDLock.Unlock()
	return s.nextFreePkgID
}

func (s *idManager) ReleasePacketID(id uint16) {
	s.pkgIDLock.Lock()
	delete(s.inFlight, id)
	s.pkgIDLock.Unlock()
}

func (s *idManager) MarshalToJSON(w io.Writer) {
	var (
		nf  uint16
		inf []uint16
	)

	// take a snapshot of things in flight
	s.pkgIDLock.Lock()
	nf = s.nextFreePkgID
	inf = make([]uint16, len(s.inFlight))
	i := 0
	for k := range s.inFlight {
		inf[i] = k
		i++
	}
	s.pkgIDLock.Unlock()

	pio.WriteString(`{"next":`, w)
	pio.WriteInt(int64(nf), w)
	if len(inf) > 0 {
		pio.WriteString(`,"inFlight":[`, w)
		for i := range inf {
			if i > 0 {
				pio.WriteByte(',', w)
			}
			pio.WriteInt(int64(inf[i]), w)
		}
		pio.WriteByte(']', w)
	}
	pio.WriteByte('}', w)
}

func (s *idManager) UnmarshalFromJSON(js *json.Decoder, t json.Token) {
	jsonstream.AssertDelimToken(t, '{')
	for {
		k, ok := jsonstream.AssertStringOrEnd(js, '}')
		if !ok {
			break
		}
		switch k {
		case "next":
			s.nextFreePkgID = uint16(jsonstream.AssertInt(js))
		case "inFlight":
			jsonstream.AssertDelim(js, '[')
			for {
				i, ok := jsonstream.AssertIntOrEnd(js, ']')
				if !ok {
					break
				}
				s.inFlight[uint16(i)] = true
			}
		}
	}
}
