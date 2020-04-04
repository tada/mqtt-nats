package bridge

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tada/mqtt-nats/mqtt/pkg"
)

type ReplyTopic struct {
	s string
	c string
	p uint16
	f byte
}

func NewReplyTopic(s Session, pp *pkg.Publish) *ReplyTopic {
	return &ReplyTopic{c: s.ClientID(), s: s.ID(), p: pp.ID(), f: pp.Flags()}
}

func ParseReplyTopic(s string) *ReplyTopic {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
		}
	}()
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

func (r *ReplyTopic) ClientID() string {
	return r.c
}

func (r *ReplyTopic) SessionID() string {
	return r.s
}

func (r *ReplyTopic) PacketID() uint16 {
	return r.p
}

func (r *ReplyTopic) Flags() byte {
	return r.f
}

func (r *ReplyTopic) String() string {
	return fmt.Sprintf("_INBOX.%s.%s.%d.%d", r.c, r.s, r.p, r.f)
}
