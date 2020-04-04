package bridge

import (
	"errors"

	"github.com/nats-io/nats.go"
	"github.com/tada/mqtt-nats/mqtt"
	"github.com/tada/mqtt-nats/mqtt/pkg"
)

func (c *client) natsPublish(pp *pkg.Publish) error {
	var err error
	if pp.IsDup() {
		if c.session.AwaitsAck(pp.ID()) {
			// Already waiting for this one
			return nil
		}
	}

	natsSubject := mqtt.ToNATS(pp.TopicName())
	switch pp.QoSLevel() {
	case 0:
		// Fire and forget
		err = c.natsConn.Publish(natsSubject, pp.Payload())
	case 1:
		// use client id and package id to form a reply subject
		replyTo := NewReplyTopic(c.session, pp).String()
		var sub *nats.Subscription
		sub, err = c.natsSubscribeAck(replyTo)
		if err == nil {
			c.session.AckRequested(pp.ID(), sub)
			err = c.natsConn.PublishRequest(natsSubject, replyTo, pp.Payload())
		}
	case 2:
		err = errors.New("QoS level 2 is not supported")
	default:
		err = errors.New("invalid QoS level")
	}
	return err
}

func (c *client) natsSubscribeAck(topic string) (*nats.Subscription, error) {
	return c.natsConn.Subscribe(topic, func(m *nats.Msg) {
		// Client may have disconnected at this point which is why it is essential to ask
		// the session manager for the session based on the replyTo subject
		mt := ParseReplyTopic(m.Subject)
		if mt != nil {
			if s := c.server.SessionManager().Get(mt.ClientID()); s != nil && s.ID() == mt.SessionID() {
				s.AckReceived(mt.PackageID())
				c.queueForWrite(pkg.PubAck(mt.PackageID()))
			}
		}
	})
}

func (c *client) addNatsSubscription(s *nats.Subscription) {
	c.subLock.Lock()
	old := c.natsSubs[s.Subject]
	c.natsSubs[s.Subject] = s
	c.subLock.Unlock()

	if old != nil {
		if err := old.Unsubscribe(); err != nil {
			c.Error(err)
		}
	}
}

func (c *client) natsSubscribe(sp *pkg.Subscribe) error {
	tps := sp.Topics()
	sps := make([]*nats.Subscription, len(tps))
	for i := range tps {
		tp := tps[i]
		sp, err := c.natsConn.Subscribe(mqtt.ToNATSSubscription(tp.Name), func(m *nats.Msg) {
			c.natsResponse(tp.QoS, m)
		})
		c.addNatsSubscription(sp)
		if err != nil {
			return err
		}
		sps[i] = sp
	}

	topicReturns := make([]byte, len(tps))
	for i := range tps {
		tr := tps[i].QoS
		if tr > 1 {
			// Max QoS level is 1.
			tr = 1
		}
		topicReturns[i] = tr
	}
	c.queueForWrite(pkg.NewSubAck(sp.ID(), topicReturns...))
	return nil
}

func (c *client) natsUnsubscribe(up *pkg.Unsubscribe) {
	tps := up.Topics()
	sps := make([]*nats.Subscription, 0, len(tps))
	c.subLock.Lock()
	for i := range tps {
		subj := mqtt.ToNATSSubscription(tps[i])
		if sp := c.natsSubs[subj]; sp != nil {
			sps = append(sps, sp)
			delete(c.natsSubs, subj)
		}
	}
	c.subLock.Unlock()
	for i := range sps {
		_ = sps[i].Unsubscribe()
	}
	c.queueForWrite(pkg.UnsubAck(up.ID()))
}

func (c *client) natsResponse(desiredQoS byte, m *nats.Msg) {
	id := uint16(0)
	flags := byte(0)
	if desiredQoS > 0 && m.Reply != `` {
		if mt := ParseReplyTopic(m.Reply); mt != nil {
			id = mt.PackageID()
			flags = mt.Flags()
		} else {
			id = c.server.NextFreePackageID()
			flags = 2 // QoS level 1
		}
	}
	pp := pkg.NewPublish(id, mqtt.FromNATS(m.Subject), flags, m.Data, false, m.Reply)
	qos := desiredQoS
	if pp.QoSLevel() < qos {
		qos = pp.QoSLevel()
	}
	c.PublishResponse(qos, pp)
}
