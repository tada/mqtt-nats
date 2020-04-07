// Package mock contains mocking/simulated versions of real runtime types
// primarily used for testing.
//
package mock

import (
	"bytes"
	"io"
	"net"
	"sync"
	"time"
)

// Connection implements net.Conn and allows recording and playback.
//
// An instance must be created by calling NewConnection() and it can thereafter be
// used the same way as a net.Conn returned from net.Dial.
// In addition to the net.Conn API, the Connection supports
// RemoteRead() and RemoteWrite() for what would be the remote end of
// a net.Conn.
//
// The primary intended use case for Connection is to help with unit testing
// logic using a net.Conn.
//
// ReadDeadLine is supported (TODO: currently WriteDeadLine is ignored as all writes will succeed until
// the system is out of memory).
//
// TODO: This Connection does not have a way to simulate
// syscall.ETIMEOUT error return resulting from a TCP keepAlive
// failure.
//
// TODO: This Connection uses the deadlines for both local and remote
//
type Connection struct {
	input             bytes.Buffer
	output            bytes.Buffer
	closed            bool
	readDeadline      time.Time
	readDeadLineTimer *time.Timer
	writeDeadline     time.Time
	moreData          *sync.Cond // Cond to wait on when there is no data to read
	moreRemoteData    *sync.Cond // Cond to wait on when there is no data to read at remote end
}

// NewConnection returns a new connection - i.e. comparable to net.Dial() but everything is hardcoded
func NewConnection() *Connection {
	// start out with a stopped timer - this timer is reset when deadline is changed
	mc := &Connection{
		moreData:          sync.NewCond(&sync.Mutex{}),
		moreRemoteData:    sync.NewCond(&sync.Mutex{}),
		readDeadLineTimer: newStoppedTimer(),
	}

	// Run a readDeadLine listener that wakes up those that are blocked reading
	go func() {
		for {
			// wait for timer to fire
			<-mc.readDeadLineTimer.C
			// timer fired, lock and Broadcast to those in blocking read
			// TODO: Fix debug level logging
			// if log.IsLevelEnabled(log.DebugLevel) {
			// 	log.Debug("mock.Connection ReadDeadline fired")
			// }
			md := mc.moreData
			md.L.Lock()
			md.Broadcast()
			md.L.Unlock()
		}
	}()
	return mc
}

// newStoppedTimer returns a stopped timer
func newStoppedTimer() *time.Timer {
	timer := time.NewTimer(1 * time.Second)
	if !timer.Stop() {
		<-timer.C // drain if it fired (very unlikely, but happens when stepping over this in debugging)
	}
	return timer
}

// Addr implements net.Addr interface and is a staic "tcp" "0.0.0.0"
type Addr struct{}

// Network returns a static "tcp" for MockConnectionAddr
func (a *Addr) Network() string { return "tcp" }

// String returns a static "0.0.0.0" for MockConnectionAddr
func (a *Addr) String() string { return "0.0.0.0" }

// Read reads data from the connection.
// Read can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetReadDeadline.
func (c *Connection) Read(b []byte) (n int, err error) {
	return c.readBufWithLock(b, &c.input, c.moreData)
}

// RemoteRead reads data from the connection's remote end (this returns what was written with Write)
func (c *Connection) RemoteRead(b []byte) (n int, err error) {
	return c.readBufWithLock(b, &c.output, c.moreRemoteData)
}

// readBufWithLock reads from the buffer and waits for more data if it is empty
func (c *Connection) readBufWithLock(b []byte, buffer *bytes.Buffer, condition *sync.Cond) (n int, err error) {
	// TODO: timeout & read of 0 bytes?
	for {
		condition.L.Lock()
	again:
		availBytes := buffer.Len()
		if c.closed && availBytes == 0 {
			condition.L.Unlock()
			return 0, io.EOF
		}
		deadline := c.readDeadline
		if !deadline.IsZero() && time.Now().After(deadline) { // while all times are after "zero time", this avoids a system call to Now()
			condition.L.Unlock()
			return 0, ErrTimeout
		}
		if availBytes == 0 {
			condition.Wait()
			goto again
		}
		n, err = buffer.Read(b)
		condition.L.Unlock()
		return n, err
	}
}

// Write writes data to the connection.
// Write can be made to time out and return an Error with Timeout() == true
// after a fixed time limit; see SetDeadline and SetWriteDeadline.
func (c *Connection) Write(b []byte) (n int, err error) {
	return c.writeBufWithLock(b, &c.output, c.moreRemoteData, c.writeDeadline)
}

// RemoteWrite writes data to the connection as if something was written on the remote side of
// a connection. (This data will be returned from subsequent Read() operations).
//
func (c *Connection) RemoteWrite(b []byte) (n int, err error) {
	return c.writeBufWithLock(b, &c.input, c.moreData, c.writeDeadline)
}

// writeBufWithLock writes to the buffer, and signals a condition if buffer goes from empty to having content
func (c *Connection) writeBufWithLock(b []byte, buffer *bytes.Buffer, condition *sync.Cond, deadline time.Time) (n int, err error) {
	condition.L.Lock()
	defer condition.L.Unlock()

	if c.closed {
		return 0, io.EOF
	}

	if !deadline.IsZero() && time.Now().After(deadline) { // while all times are after "zero time", this avoids a system call to Now()
		return 0, ErrTimeout
	}

	availBytes := buffer.Len()
	n, err = buffer.Write(b)
	if availBytes == 0 {
		condition.Broadcast()
	}
	return n, err
}

// Close closes the connection.
// Any blocked Read or Write operations will be unblocked and return errors.
func (c *Connection) Close() error {
	c.moreRemoteData.L.Lock()
	defer c.moreRemoteData.L.Unlock()
	c.moreData.L.Lock()
	defer c.moreData.L.Unlock()

	// mark closed
	c.closed = true

	// release blocked readers so they pick up the close
	if c.input.Len() == 0 {
		c.moreData.Broadcast()
	}
	// release blocked remote readers so they pick up the close
	if c.output.Len() == 0 {
		c.moreRemoteData.Broadcast()
	}
	return nil
}

// LocalAddr returns a hardcoded local network address.
func (c *Connection) LocalAddr() net.Addr {
	return &Addr{}
}

// RemoteAddr returns a hardcoded remote network address.
func (c *Connection) RemoteAddr() net.Addr {
	return &Addr{}
}

// SetDeadline sets the read and write deadlines associated
// with the connection. It is equivalent to calling both
// SetReadDeadline and SetWriteDeadline.
//
// A deadline is an absolute time after which I/O operations
// fail with a timeout (see type Error) instead of
// blocking. The deadline applies to all future and pending
// I/O, not just the immediately following call to Read or
// Write. After a deadline has been exceeded, the connection
// can be refreshed by setting a deadline in the future.
//
// An idle timeout can be implemented by repeatedly extending
// the deadline after successful Read or Write calls.
//
// A zero value for t means I/O operations will not time out.
//
// Note that if a TCP connection has keep-alive turned on,
// which is the default unless overridden by Dialer.KeepAlive
// or ListenConfig.KeepAlive, then a keep-alive failure may
// also return a timeout error. On Unix systems a keep-alive
// failure on I/O can be detected using
// errors.Is(err, syscall.ETIMEDOUT).
//
// TODO: This MockConnection does not have a way to simulate
// syscall.ETIMEOUT error return resulting from a TCP keepAlive
// failure.
//
// TODO: At present the WriteDeadLine does not have any effect.
func (c *Connection) SetDeadline(t time.Time) error {
	if err := c.SetReadDeadline(t); err != nil {
		return err
	}
	if err := c.SetWriteDeadline(t); err != nil {
		return err
	}
	return nil
}

// SetReadDeadline sets the deadline for future Read calls
// and any currently-blocked Read call.
// A zero value for t means Read will not time out.
func (c *Connection) SetReadDeadline(t time.Time) error {
	c.moreData.L.Lock()
	defer c.moreData.L.Unlock()
	c.readDeadline = t

	// stop the currently ticking (or possibly fired) timer
	oldTimer := c.readDeadLineTimer
	if !oldTimer.Stop() {
		select {
		case <-oldTimer.C: // drain if it fired
		default: // it was stopped already
		}
	}
	if t.IsZero() {
		// No need to wake those that are blocked, simply keep the timer in stopped state
		return nil
	}
	// Set a new duration
	oldTimer.Reset(time.Until(t))
	return nil
}

// SetWriteDeadline sets the deadline for future Write calls
// and any currently-blocked Write call.
// Even if write times out, it may return n > 0, indicating that
// some of the data was successfully written.
// A zero value for t means Write will not time out.
// TODO: At present this WriteDeadLine does not have any effect.
func (c *Connection) SetWriteDeadline(t time.Time) error {
	c.writeDeadline = t
	return nil
}

// RemoteIO is an interface for operations on what MockConnection.Remote() returns
type RemoteIO interface {
	io.ReadWriter
	io.ByteReader
}

// Remote is a io.ReadWriter that adapts the "remote" end of a MockConnection to
// io.ReadWriter
type Remote struct {
	conn *Connection
}

func (r *Remote) Read(b []byte) (int, error) {
	return r.conn.RemoteRead(b)
}

func (r *Remote) Write(b []byte) (int, error) {
	return r.conn.RemoteWrite(b)
}

// ReadByte reads one byte - implements io.ByteReader
func (r *Remote) ReadByte() (byte, error) {
	b := make([]byte, 1)
	n, err := r.conn.RemoteRead(b)
	if err != nil || n != 1 {
		return 0, err
	}
	return b[0], nil
}

// Remote returns a io.ReadWriter for the remote end of this MockConnetion
func (c *Connection) Remote() RemoteIO {
	return &Remote{conn: c}
}

// TimeoutError is returned for an expired deadline.
// It implements net.Error interface
// This type is needed because the error actually returned from net.Conn is an internal data type
//
type TimeoutError struct{}

// Error Implement the net.Error interface.
func (e *TimeoutError) Error() string { return "i/o timeout" }

// Timeout Implement the net.Error interface.
func (e *TimeoutError) Timeout() bool { return true }

// Temporary Implement the net.Error interface.
func (e *TimeoutError) Temporary() bool { return true }

// ErrTimeout is returned for an expired deadline
var ErrTimeout error = &TimeoutError{}
