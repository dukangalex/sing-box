package chain

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"testing"
	"time"

	M "github.com/sagernet/sing/common/metadata"
)

type testDialer struct {
	network     string
	destination M.Socksaddr
	conn        net.Conn
	err         error
}

func (d *testDialer) DialContext(_ context.Context, network string, destination M.Socksaddr) (net.Conn, error) {
	d.network = network
	d.destination = destination
	return d.conn, d.err
}

func (d *testDialer) ListenPacket(context.Context, M.Socksaddr) (net.PacketConn, error) {
	return nil, errors.New("not implemented")
}

func TestTransportDialRequiresEntryAndLanding(t *testing.T) {
	if _, err := (Transport{}).Dial(context.Background(), "tcp"); err == nil {
		t.Fatal("expected nil entry error")
	}
	d := &testDialer{}
	if _, err := (Transport{Entry: d}).Dial(context.Background(), "tcp"); err == nil {
		t.Fatal("expected empty landing error")
	}
}

func TestTransportRejectsNonTCP(t *testing.T) {
	d := &testDialer{}
	addr := M.SocksaddrFrom(netip.MustParseAddr("192.0.2.10"), 443)
	if _, err := (Transport{Entry: d, Landing: addr}).Dial(context.Background(), "udp"); err == nil {
		t.Fatal("expected UDP rejection")
	}
}

func TestTransportDialsLandingThroughEntry(t *testing.T) {
	d := &testDialer{conn: &fakeConn{}}
	addr := M.SocksaddrFrom(netip.MustParseAddr("192.0.2.10"), 443)
	conn, err := (Transport{Entry: d, Landing: addr}).Dial(context.Background(), "tcp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if conn == nil {
		t.Fatal("expected connection")
	}
	if d.destination != addr {
		t.Fatalf("destination = %v, want %v", d.destination, addr)
	}
}

func TestTransportDoesNotFallbackOnLandingDialFailure(t *testing.T) {
	d := &testDialer{err: errors.New("landing unavailable")}
	addr := M.SocksaddrFrom(netip.MustParseAddr("192.0.2.10"), 443)
	if _, err := (Transport{Entry: d, Landing: addr}).Dial(context.Background(), "tcp"); err == nil {
		t.Fatal("expected strict failure")
	}
}

type fakeConn struct{}

func (*fakeConn) Read([]byte) (int, error)           { return 0, nil }
func (*fakeConn) Write(p []byte) (int, error)        { return len(p), nil }
func (*fakeConn) Close() error                       { return nil }
func (*fakeConn) LocalAddr() net.Addr                { return &net.TCPAddr{} }
func (*fakeConn) RemoteAddr() net.Addr               { return &net.TCPAddr{} }
func (*fakeConn) SetDeadline(_ time.Time) error      { return nil }
func (*fakeConn) SetReadDeadline(_ time.Time) error  { return nil }
func (*fakeConn) SetWriteDeadline(_ time.Time) error { return nil }
