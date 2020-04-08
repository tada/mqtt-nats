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
		// use client id and packet id to form a reply subject
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
				c.cancelNatsSubscriptions(s.AckReceived(mt.PacketID()))
				c.queueForWrite(pkg.PubAck(mt.PacketID()))
			}
		}
	})
}

func (c *client) cancelNatsSubscriptions(nss []*nats.Subscription) {
	for i := range nss {
		ns := nss[i]
		if err := ns.Unsubscribe(); err != nil {
			c.Error("NATS unsubscribe", ns.Subject, err)
		}
	}
}

func (c *client) natsSubscribe(sp *pkg.Subscribe) {
	tps := sp.Topics()
	nms := make([]string, len(tps))
	qss := make([]byte, len(tps))
	var nss []*nats.Subscription
	c.subLock.Lock()
	for i := range tps {
		tp := tps[i]
		nm := mqtt.ToNATSSubscription(tp.Name)
		nms[i] = nm
		qss[i] = tp.QoS
		if os := c.natsSubs[nm]; os != nil {
			delete(c.natsSubs, nm)
			nss = append(nss, os)
		}
	}
	c.subLock.Unlock()
	c.cancelNatsSubscriptions(nss)

	nss = make([]*nats.Subscription, 0, len(nms))
	for i := range nms {
		nm := nms[i]
		qs := qss[i]
		if qs > 1 {
			qs = 1
			qss[i] = 1
		}
		ns, err := c.natsConn.Subscribe(nm, func(m *nats.Msg) {
			c.natsResponse(qs, m)
		})
		if err == nil {
			nss = append(nss, ns)
		} else {
			c.Error("NATS subscribe", nm, err)
			qss[i] = 0x80
		}
	}
	c.subLock.Lock()
	for i := range nss {
		ns := nss[i]
		c.natsSubs[ns.Subject] = ns
	}
	c.subLock.Unlock()
	c.queueForWrite(pkg.NewSubAck(sp.ID(), qss...))
}

func (c *client) natsUnsubscribe(up *pkg.Unsubscribe) {
	tps := up.Topics()
	nss := make([]*nats.Subscription, 0, len(tps))
	c.subLock.Lock()
	for i := range tps {
		subj := mqtt.ToNATSSubscription(tps[i])
		if ns := c.natsSubs[subj]; ns != nil {
			nss = append(nss, ns)
			delete(c.natsSubs, subj)
		}
	}
	c.subLock.Unlock()
	c.cancelNatsSubscriptions(nss)
	c.queueForWrite(pkg.UnsubAck(up.ID()))
}

func (c *client) natsResponse(desiredQoS byte, m *nats.Msg) {
	id := uint16(0)
	flags := byte(0)
	if desiredQoS > 0 && m.Reply != `` {
		if mt := ParseReplyTopic(m.Reply); mt != nil {
			id = mt.PacketID()
			flags = mt.Flags()
		} else {
			id = c.server.NextFreePacketID()
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
