package pkg_test

import (
	"bytes"
	"testing"

	"github.com/tada/mqtt-nats/test/utils"

	"github.com/tada/mqtt-nats/mqtt"

	"github.com/tada/mqtt-nats/mqtt/pkg"
)

func TestParseConnect(t *testing.T) {
	c1 := pkg.NewConnect(`cid`, true, 5, &pkg.Will{
		Topic:   "my/will",
		Message: []byte("the will"),
		QoS:     1,
		Retain:  false,
	}, &pkg.Credentials{User: "bob", Password: []byte("password")})
	writeReadAndCompare(t, c1, "CONNECT (c1, k5, u1, p1, w(r0, q1, 'my/will', ... (8 bytes)))")
}

func TestParseConnect_badLen(t *testing.T) {
	_, err := pkg.ParseConnect(mqtt.NewReader(bytes.NewReader([]byte{})), pkg.TpConnAck, 20)
	utils.CheckError(err, t)
}

func TestParseConnect_badProto(t *testing.T) {
	w := mqtt.NewWriter()
	w.WriteU16(5)
	w.WriteU8('N')
	w.WriteU8('O')
	w.WriteU8('N')
	w.WriteU8('O')
	_, err := pkg.ParseConnect(mqtt.NewReader(bytes.NewReader(w.Bytes())), pkg.TpConnAck, 6)
	utils.CheckError(err, t)
}

func TestParseConnect_illegalProto(t *testing.T) {
	w := mqtt.NewWriter()
	w.WriteString("NONO")
	_, err := pkg.ParseConnect(mqtt.NewReader(bytes.NewReader(w.Bytes())), pkg.TpConnAck, 6)
	utils.CheckError(err, t)
}

func TestParseConnect_badClientLevel(t *testing.T) {
	w := mqtt.NewWriter()
	w.WriteString("MQTT")
	_, err := pkg.ParseConnect(mqtt.NewReader(bytes.NewReader(w.Bytes())), pkg.TpConnAck, 6)
	utils.CheckError(err, t)
}

func TestParseConnect_illegalClientLevel(t *testing.T) {
	w := mqtt.NewWriter()
	w.WriteString("MQTT")
	w.WriteU8(2)
	_, err := pkg.ParseConnect(mqtt.NewReader(bytes.NewReader(w.Bytes())), pkg.TpConnAck, 7)
	utils.CheckError(err, t)
}

func TestParseConnect_badConnectFlags(t *testing.T) {
	w := mqtt.NewWriter()
	w.WriteString("MQTT")
	w.WriteU8(4)
	_, err := pkg.ParseConnect(mqtt.NewReader(bytes.NewReader(w.Bytes())), pkg.TpConnAck, 7)
	utils.CheckError(err, t)
}

func TestParseConnect_badKeepAlive(t *testing.T) {
	w := mqtt.NewWriter()
	w.WriteString("MQTT")
	w.WriteU8(4)
	w.WriteU8(0)
	_, err := pkg.ParseConnect(mqtt.NewReader(bytes.NewReader(w.Bytes())), pkg.TpConnAck, 8)
	utils.CheckError(err, t)
}

func TestParseConnect_badClientID(t *testing.T) {
	w := mqtt.NewWriter()
	w.WriteString("MQTT")
	w.WriteU8(4)
	w.WriteU8(0)
	w.WriteU16(5)
	_, err := pkg.ParseConnect(mqtt.NewReader(bytes.NewReader(w.Bytes())), pkg.TpConnAck, 10)
	utils.CheckError(err, t)
}

func TestParseConnect_badWillTopic(t *testing.T) {
	w := mqtt.NewWriter()
	w.WriteString("MQTT")
	w.WriteU8(4)
	w.WriteU8(0x04)
	w.WriteU16(5)
	w.WriteString("cid")
	_, err := pkg.ParseConnect(mqtt.NewReader(bytes.NewReader(w.Bytes())), pkg.TpConnAck, 15)
	utils.CheckError(err, t)
}

func TestParseConnect_badWillMessage(t *testing.T) {
	w := mqtt.NewWriter()
	w.WriteString("MQTT")
	w.WriteU8(4)
	w.WriteU8(0x04)
	w.WriteU16(5)
	w.WriteString("cid")
	w.WriteString("wtp")
	_, err := pkg.ParseConnect(mqtt.NewReader(bytes.NewReader(w.Bytes())), pkg.TpConnAck, 20)
	utils.CheckError(err, t)
}

func TestParseConnect_badUser(t *testing.T) {
	w := mqtt.NewWriter()
	w.WriteString("MQTT")
	w.WriteU8(4)
	w.WriteU8(0x80)
	w.WriteU16(5)
	w.WriteString("cid")
	_, err := pkg.ParseConnect(mqtt.NewReader(bytes.NewReader(w.Bytes())), pkg.TpConnAck, 15)
	utils.CheckError(err, t)
}

func TestParseConnect_badPw(t *testing.T) {
	w := mqtt.NewWriter()
	w.WriteString("MQTT")
	w.WriteU8(4)
	w.WriteU8(0x40)
	w.WriteU16(5)
	w.WriteString("cid")
	_, err := pkg.ParseConnect(mqtt.NewReader(bytes.NewReader(w.Bytes())), pkg.TpConnAck, 15)
	utils.CheckError(err, t)
}

func TestParseConnAck(t *testing.T) {
	writeReadAndCompare(t, pkg.NewConnAck(false, 1), "CONNACK (s0, rt1)")
}

func TestParseConnAck_badLen(t *testing.T) {
	_, err := pkg.ParseConnAck(mqtt.NewReader(bytes.NewReader([]byte{})), pkg.TpConnAck, 0)
	utils.CheckError(err, t)
}

func TestParseConnAck_badBytes(t *testing.T) {
	_, err := pkg.ParseConnAck(mqtt.NewReader(bytes.NewReader([]byte{1})), pkg.TpConnAck, 2)
	utils.CheckError(err, t)
}

func TestParseDisconnect(t *testing.T) {
	writeReadAndCompare(t, pkg.DisconnectSingleton, "DISCONNECT")
}

func TestReturnCode_Error(t *testing.T) {
	tests := []struct {
		name string
		code pkg.ReturnCode
		want string
	}{
		{
			name: "RtAccepted",
			code: pkg.RtAccepted,
			want: "accepted",
		},
		{
			name: "RtUnacceptableProtocolVersion",
			code: pkg.RtUnacceptableProtocolVersion,
			want: "unacceptable protocol version",
		},
		{
			name: "RtIdentifierRejected",
			code: pkg.RtIdentifierRejected,
			want: "identifier rejected",
		},
		{
			name: "RtServerUnavailable",
			code: pkg.RtServerUnavailable,
			want: "server unavailable",
		},
		{
			name: "RtBadUserNameOrPassword",
			code: pkg.RtBadUserNameOrPassword,
			want: "bad user name or password",
		},
		{
			name: "RtNotAuthorized",
			code: pkg.RtNotAuthorized,
			want: "not authorized",
		},
		{
			name: "RtNotAuthorized",
			code: pkg.ReturnCode(99),
			want: "unknown error",
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.code.Error(); got != tt.want {
				t.Errorf("ReturnCode.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}
