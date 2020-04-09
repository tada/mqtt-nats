package mock

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/tada/mqtt-nats/test/utils"
)

func Test_MockConnection_implements_net_Conn(t *testing.T) {
	defer utils.ShouldNotPanic(t)
	_ = net.Conn(NewConnection())
}

func Test_MockConnection_has_hardocded_local_and_remote_addr(t *testing.T) {
	conn := NewConnection()
	local := conn.LocalAddr()
	remote := conn.RemoteAddr()
	utils.CheckEqual("0.0.0.0", local.String(), t)
	utils.CheckEqual("tcp", local.Network(), t)
	utils.CheckEqual("0.0.0.0", remote.String(), t)
	utils.CheckEqual("tcp", remote.Network(), t)
}

// Test that RemoteWrite can be called without error
func Test_MockConnection_Can_do_RemoteWrite(t *testing.T) {
	conn := NewConnection()
	n, err := conn.RemoteWrite([]byte("test"))
	utils.CheckEqual(4, n, t)
	utils.CheckNotError(err, t)
}

// Test that what is written "remotely" can be read "locally"
func Test_MockConnection_Read_reads_what_was_written_with_RemoteWrite(t *testing.T) {
	conn := NewConnection()
	n, err := conn.RemoteWrite([]byte("test"))
	utils.CheckEqual(4, n, t)
	utils.CheckNotError(err, t)
	buf := make([]byte, 4)
	n, err = conn.Read(buf)
	utils.CheckEqual(4, n, t)
	utils.CheckNotError(err, t)
}

// Test that what is written "locally" can be read "remotely"
func Test_MockConnection_ReadRemote_reads_what_was_written_with_Write(t *testing.T) {
	conn := NewConnection()
	n, err := conn.Write([]byte("test"))
	utils.CheckEqual(4, n, t)
	utils.CheckNotError(err, t)
	buf := make([]byte, 4)
	n, err = conn.RemoteRead(buf)
	utils.CheckEqual(4, n, t)
	utils.CheckNotError(err, t)
}

func Test_MockConnection_Read_waits_for_data_until_close(t *testing.T) {
	conn := NewConnection()
	readResult := make(chan error)
	timeout := make(chan bool)
	readEndTime := time.Now()
	closeTime := time.Now().Add(1 * time.Millisecond) // ensure this is after

	go func() {
		delay := 200 * time.Millisecond
		time.Sleep(delay)
		timeout <- true
	}()
	go func() {
		aByte := make([]byte, 1)
		_, err := conn.Read(aByte)
		readEndTime = time.Now()
		readResult <- err
	}()

	// Wait for the timeout
	<-timeout
	if err := conn.Close(); err != nil {
		panic(err)
	}

	// Wait for the read
	err := <-readResult

	// Did they occur in the expected order?
	utils.CheckTrue(readEndTime.After(closeTime), t)
	utils.CheckTrue(err == io.EOF, t)
}

func Test_MockConnection_Read_waits_until_given_read_deadline(t *testing.T) {
	conn := NewConnection()
	if err := conn.SetDeadline(time.Now().Add(200 * time.Millisecond)); err != nil {
		t.Fatal(err)
	}

	aByte := make([]byte, 1)
	_, err := conn.Read(aByte)
	nerr, ok := err.(net.Error)
	if !ok {
		t.Fatalf("Expected a net.Error but could not convert it")
	}
	utils.CheckTrue(nerr.Timeout(), t)
}

func Test_MockConnection_Read_returns_amount_read_if_buffer_becomes_empty(t *testing.T) {
	conn := NewConnection()
	oneByte := []byte{1}
	n, err := conn.RemoteWrite(oneByte)
	utils.CheckEqual(1, n, t)
	utils.CheckNotError(err, t)

	threeBytes := make([]byte, 3)
	n, err = conn.Read(threeBytes)
	utils.CheckEqual(1, n, t)
	utils.CheckNotError(err, t)
	utils.CheckEqual(byte(1), threeBytes[0], t)

	oneByte = []byte{2}
	n, err = conn.RemoteWrite(oneByte)
	utils.CheckEqual(1, n, t)
	utils.CheckNotError(err, t)

	n, err = conn.Read(threeBytes)
	utils.CheckEqual(1, n, t)
	utils.CheckNotError(err, t)
	utils.CheckEqual(byte(2), threeBytes[0], t)
}

// TODO: Test Multithreaded reading and writing
// 1. n threads write 1 byte each, n threads read one byte each - all bytes are read
