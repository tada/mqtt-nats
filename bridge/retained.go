package bridge

import (
	"encoding/json"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/nats-io/nats.go"
	"github.com/tada/mqtt-nats/jsonstream"
	"github.com/tada/mqtt-nats/mqtt"
	"github.com/tada/mqtt-nats/mqtt/pkg"
	"github.com/tada/mqtt-nats/pio"
)

type retained struct {
	lock  sync.RWMutex
	msgs  map[string]*pkg.Publish
	order []string
}

func (r *retained) Empty() bool {
	r.lock.RLock()
	empty := len(r.order) == 0
	r.lock.RUnlock()
	return empty
}

func (r *retained) MarshalJSON() ([]byte, error) {
	return jsonstream.Marshal(r)
}

func (r *retained) UnmarshalJSON(bs []byte) error {
	return jsonstream.Unmarshal(r, bs)
}

func (r *retained) MarshalToJSON(w io.Writer) {
	r.lock.RLock()
	defer r.lock.RUnlock()
	sep := byte('{')
	for _, t := range r.order {
		pio.WriteByte(sep, w)
		sep = byte(',')
		jsonstream.WriteString(t, w)
		pio.WriteByte(':', w)
		r.msgs[t].MarshalToJSON(w)
	}
	if sep == '{' {
		pio.WriteByte(sep, w)
	}
	pio.WriteByte('}', w)
}

func (r *retained) UnmarshalFromJSON(js *json.Decoder, t json.Token) {
	jsonstream.AssertDelimToken(t, '{')
	r.msgs = make(map[string]*pkg.Publish)
	r.order = nil
	for {
		s, ok := jsonstream.AssertStringOrEnd(js, '}')
		if !ok {
			break
		}
		p := &pkg.Publish{}
		jsonstream.AssertConsumer(js, p)
		r.msgs[s] = p
		r.order = append(r.order, s)
	}
}

func (r *retained) add(m *pkg.Publish) bool {
	r.lock.Lock()
	t := m.TopicName()
	_, present := r.msgs[t]
	r.msgs[t] = m
	if !present {
		r.order = append(r.order, t)
	}
	r.lock.Unlock()
	return !present
}

func (r *retained) drop(t string) bool {
	dropped := false
	r.lock.Lock()
	if _, present := r.msgs[t]; present {
		delete(r.msgs, t)
		o := r.order
		last := len(o) - 1
		for i := 0; i <= last; i++ {
			if o[i] == t {
				copy(o[i:], o[i+1:])
				o[last] = `` // allow GC of last
				r.order = o[:last]
				dropped = true
				break
			}
		}
	}
	r.lock.Unlock()
	return dropped
}

var topicAndQoS = regexp.MustCompile(`^(.+)/([012])$`)

func (r *retained) messagesMatchingRetainRequest(m *nats.Msg) ([]*pkg.Publish, []byte) {
	natsTopics := strings.Split(string(m.Data), ",")
	topics := make([]pkg.Topic, len(natsTopics))
	for i := range natsTopics {
		nt := natsTopics[i]
		ts := topicAndQoS.FindStringSubmatch(nt)
		tp := pkg.Topic{}
		if ts == nil {
			tp.Name = mqtt.FromNATSSubscription(nt)
		} else {
			tp.Name = mqtt.FromNATSSubscription(ts[1])
			qos, _ := strconv.Atoi(ts[2])
			tp.QoS = byte(qos)
		}
		topics[i] = tp
	}
	return r.matchingMessages(topics)
}

func (r *retained) publishMatching(s *pkg.Subscribe, c Client) {
	pps, qs := r.matchingMessages(s.Topics())
	for i := range pps {
		pp := pps[i]
		c.PublishResponse(qs[i], pp)
	}
}

func (r *retained) matchingMessages(tps []pkg.Topic) ([]*pkg.Publish, []byte) {
	tpl := len(tps)

	// Create slices of subscription regexps and desired QoS. One entry for each subscription topic
	txs := make([]*regexp.Regexp, tpl)
	dqs := make([]byte, tpl)
	for i := 0; i < tpl; i++ {
		tp := tps[i]
		txs[i] = mqtt.SubscriptionToRegexp(tp.Name)
		dqs[i] = tp.QoS
	}

	// For each subscription topic, extract matching packages and desired QoS
	pps := make([]*pkg.Publish, 0)
	qs := make([]byte, 0)

	r.lock.RLock()
	for i := 0; i < tpl; i++ {
		tx := txs[i]
		dq := dqs[i]
		for _, t := range r.order {
			if tx.MatchString(t) {
				pps = append(pps, r.msgs[t])
				qs = append(qs, dq)
			}
		}
	}
	r.lock.RUnlock()
	return pps, qs
}
