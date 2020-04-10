package bridge

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tada/mqtt-nats/mqtt/pkg"
)

// ReplyTopic represents the decoded form of the NATS reply-topic that the bridge uses to track
// messages that are in need of an ACK.
type ReplyTopic struct {
	s string
	c string
	p uint16
	f byte
}

// NewReplyTopic creates a new ReplyTopic based on a pkg.Publish packet.
func NewReplyTopic(s Session, pp *pkg.Publish) *ReplyTopic {
	return &ReplyTopic{c: s.ClientID(), s: s.ID(), p: pp.ID(), f: pp.Flags()}
}

// ParseReplyTopic creates a new ReplyTopic by parsing a NATS reply-to string
func ParseReplyTopic(s string) *ReplyTopic {
	ps := strings.Split(s, ".")
	if len(ps) == 5 && ps[0] == "_INBOX" {
		p, err := strconv.Atoi(ps[3])
		if err == nil {
			var f int
			f, err = strconv.Atoi(ps[4])
			if err == nil {
				return &ReplyTopic{c: ps[1], s: ps[2], p: uint16(p), f: byte(f)}
			}
		}
	}
	return nil
}

// ClientID returns the ID of the client where the message originated
func (r *ReplyTopic) ClientID() string {
	return r.c
}

// SessionID returns the ID of the client session within the mqtt-nats bridge
func (r *ReplyTopic) SessionID() string {
	return r.s
}

// PacketID returns the packet identifier of the original packet
func (r *ReplyTopic) PacketID() uint16 {
	return r.p
}

// Flags returns the packet flags of the original packet
func (r *ReplyTopic) Flags() byte {
	return r.f
}

// String returns the NATS string form of the reply-topic
func (r *ReplyTopic) String() string {
	return fmt.Sprintf("_INBOX.%s.%s.%d.%d", r.c, r.s, r.p, r.f)
}
