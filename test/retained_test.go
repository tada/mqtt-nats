package test

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/tada/mqtt-nats/mqtt"
	"github.com/tada/mqtt-nats/mqtt/pkg"
)

func decodeRetained(t *testing.T, data []byte) []*pkg.Publish {
	t.Helper()
	var ms []map[string]string
	err := json.Unmarshal(data, &ms)
	if err != nil {
		t.Fatal(err)
	}
	ns := make([]*pkg.Publish, len(ms))
	for i := range ms {
		m := ms[i]
		var pl []byte
		if ps, ok := m["payload"]; ok {
			pl = []byte(ps)
		} else if pe, ok := m["payloadEnc"]; ok {
			if pl, err = base64.StdEncoding.DecodeString(pe); err != nil {
				t.Fatal(err)
			}
		}
		ns[i] = pkg.NewPublish2(0, mqtt.FromNATS(m["subject"]), pl, 0, false, true)
	}
	return ns
}

func TestNATS_requestRetained(t *testing.T) {
	conn := mqttConnectClean(t, mqttPort)
	pp1 := pkg.NewPublish2(0, "testing/s.o.m.e/retained/first", []byte("the first retained message"), 0, false, true)
	pp2 := pkg.NewPublish2(0, "testing/s.o.m.e/retained/second", []byte("the second retained message"), 0, false, true)
	mqttSend(t, conn, pp1, pp2)
	mqttDisconnect(t, conn)

	nc := natsConnect(t, natsPort)
	defer nc.Close()

	m, err := nc.Request(retainedRequestTopic, []byte("testing.s/o/m/e.retained.>"), 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	pps := decodeRetained(t, m.Data)
	if !(len(pps) == 2 && pp1.Equals(pps[0]) && pp2.Equals(pps[1])) {
		t.Fatal("unexpected retained publication")
	}

	m, err = nc.Request(retainedRequestTopic, []byte("do.not.find.this"), 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	pps = decodeRetained(t, m.Data)
	if len(pps) != 0 {
		t.Fatal("unexpected retained publication")
	}
}
