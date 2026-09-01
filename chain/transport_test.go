package chain

import (
	"context"
	"errors"
	"net"
	"testing"
)

type testDialer struct {
	network     string
	destination net.Addr
	conn        net.Conn
	err         error
}

func (d *testDialer) DialContext(_ context.Context, network string, destination net.Addr) (net.Conn, error) {
	d.network = network
	d.destination = destination
	return d.conn, d.err
}

func TestTransportDialRequiresEntryAndLanding(t *testing.T) {
	addr := &net.TCPAddr{IP: net.IPv4(192, 0, 2, 10), Port: 443}
	if _, err := (Transport{Landing: addr}).Dial(context.Background(), "tcp"); err == nil {
		t.Fatal("expected nil entry error")
	}
	if _, err := (Transport{Entry: &testDialer{}}).Dial(context.Background(), "tcp"); err == nil {
		t.Fatal("expected nil landing error")
	}
}

func TestTransportRejectsNonTCP(t *testing.T) {
	d := &testDialer{}
	addr := &net.TCPAddr{IP: net.IPv4(192, 0, 2, 10), Port: 443}
	if _, err := (Transport{Entry: d, Landing: addr}).Dial(context.Background(), "udp"); err == nil {
		t.Fatal("expected UDP rejection")
	}
}

func TestTransportDialsLandingThroughEntry(t *testing.T) {
	d := &testDialer{conn: &fakeConn{}}
	addr := &net.TCPAddr{IP: net.IPv4(192, 0, 2, 10), Port: 443}
	conn, err := (Transport{Entry: d, Landing: addr}).Dial(context.Background(), "tcp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected connection")
	}
	if d.destination.String() != addr.String() {
		t.Fatalf("destination = %s, want %s", d.destination, addr)
	}
}

func TestTransportDoesNotFallbackOnLandingDialFailure(t *testing.T) {
	d := &testDialer{err: errors.New("landing unavailable")}
	addr := &net.TCPAddr{IP: net.IPv4(192, 0, 2, 10), Port: 443}
	if _, err := (Transport{Entry: d, Landing: addr}).Dial(context.Background(), "tcp"); err == nil {
		t.Fatal("expected strict failure")
	}
}

type fakeConn struct{}

func (*fakeConn) Read([]byte) (int, error)         { return 0, nil }
func (*fakeConn) Write(p []byte) (int, error)      { return len(p), nil }
func (*fakeConn) Close() error                     { return nil }
func (*fakeConn) LocalAddr() net.Addr              { return &net.TCPAddr{} }
func (*fakeConn) RemoteAddr() net.Addr             { return &net.TCPAddr{} }
func (*fakeConn) SetDeadline(_ time.Time) error    { return nil }
func (*fakeConn) SetReadDeadline(_ time.Time) error { return nil }
func (*fakeConn) SetWriteDeadline(_ time.Time) error { return nil }
