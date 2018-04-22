package rpcext

import (
	"bufio"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"net/url"
	"time"
)

var connected = "200 Connected to Go RPC"

var DefaultNetDialer = &net.Dialer{
	Timeout:   30 * time.Second,
	KeepAlive: 30 * time.Second,
	DualStack: true,
}

var DefaultDialer = &Dialer{}

type Dialer struct {
	HTTPClient *http.Client // http.DefaultClient if nil
	NetDialer  *net.Dialer  // DefaultDialer if nil
}

func DialHTTP(endpoint string) (*rpc.Client, error) {
	return DefaultDialer.DialHTTP(endpoint)
}

func (d *Dialer) DialHTTP(endpoint string) (*rpc.Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("CONNECT", u.String(), nil)
	if err != nil {
		return nil, err
	}
	conn, err := d.netDialer().Dial("tcp", u.Host)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "http", "":
		conn, err = d.connectHTTP(conn, req)
	case "https":
		conn, err = d.connectHTTPS(conn, req)
	default:
		err = fmt.Errorf("scheme %q is not supported", u.Scheme)
	}
	if err != nil {
		return nil, err
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), req)
	if err == nil && resp.Status == connected {
		return rpc.NewClient(conn), nil
	}
	if err == nil {
		err = errors.New("unexpected HTTP response: " + resp.Status)
	}
	conn.Close()
	return nil, &net.OpError{
		Op:   "dial-http",
		Net:  u.String(),
		Addr: nil,
		Err:  err,
	}
}

func (d *Dialer) connectHTTP(conn net.Conn, req *http.Request) (net.Conn, error) {
	if err := req.Write(conn); err != nil {
		return nil, err
	}
	return conn, nil
}

func (d *Dialer) connectHTTPS(conn net.Conn, req *http.Request) (net.Conn, error) {
	conn = tls.Client(conn, &tls.Config{
		InsecureSkipVerify: false,
	})
	return d.connectHTTP(conn, req)
}

func (d *Dialer) httpClient() *http.Client {
	if d.HTTPClient != nil {
		return d.HTTPClient
	}
	return http.DefaultClient
}

func (d *Dialer) netDialer() *net.Dialer {
	if d.NetDialer != nil {
		return d.NetDialer
	}
	return DefaultNetDialer
}
