package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"

	"tailscale.com/tsnet"
)

const (
	SOCKS5_VERSION = 0x05
	NO_AUTH        = 0x00
	CONNECT        = 0x01
	IPV4           = 0x01
	DOMAIN         = 0x03
	IPV6           = 0x04
	SUCCESS        = 0x00
	FAILURE        = 0x01
)

type SOCKS5Proxy struct {
	server *tsnet.Server
}

func NewSOCKS5Proxy(hostname, authkey string) *SOCKS5Proxy {
	s := &tsnet.Server{
		Hostname: hostname,
		AuthKey:  authkey,
	}
	return &SOCKS5Proxy{server: s}
}

func (p *SOCKS5Proxy) Start(port string) error {
	listener, err := p.server.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	defer listener.Close()

	log.Printf("SOCKS5 proxy listening on Tailscale network at %s:%s", p.server.Hostname, port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}
		go p.handleConnection(conn)
	}
}

func (p *SOCKS5Proxy) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Step 1: Handle authentication negotiation
	if err := p.handleAuth(conn); err != nil {
		log.Printf("Auth failed: %v", err)
		return
	}

	// Step 2: Handle CONNECT request
	target, err := p.handleConnect(conn)
	if err != nil {
		log.Printf("Connect failed: %v", err)
		return
	}

	// Step 3: Establish connection to target through Tailscale
	targetConn, err := p.server.Dial(context.Background(), "tcp", target)
	if err != nil {
		log.Printf("Failed to connect to target %s: %v", target, err)
		p.sendConnectResponse(conn, FAILURE, "0.0.0.0", "0")
		return
	}
	defer targetConn.Close()

	// Send success response
	if err := p.sendConnectResponse(conn, SUCCESS, "0.0.0.0", "0"); err != nil {
		log.Printf("Failed to send success response: %v", err)
		return
	}

	// Step 4: Relay data between client and target
	p.relay(conn, targetConn)
}

func (p *SOCKS5Proxy) handleAuth(conn net.Conn) error {
	buf := make([]byte, 2)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return err
	}

	version := buf[0]
	nMethods := buf[1]

	if version != SOCKS5_VERSION {
		return fmt.Errorf("unsupported SOCKS version: %d", version)
	}

	methods := make([]byte, nMethods)
	if _, err := io.ReadFull(conn, methods); err != nil {
		return err
	}

	// Check if NO_AUTH is supported
	noAuthSupported := false
	for _, method := range methods {
		if method == NO_AUTH {
			noAuthSupported = true
			break
		}
	}

	response := []byte{SOCKS5_VERSION, NO_AUTH}
	if !noAuthSupported {
		response[1] = 0xFF // No acceptable methods
		conn.Write(response)
		return fmt.Errorf("no acceptable authentication methods")
	}

	_, err := conn.Write(response)
	return err
}

func (p *SOCKS5Proxy) handleConnect(conn net.Conn) (string, error) {
	buf := make([]byte, 4)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return "", err
	}

	version := buf[0]
	cmd := buf[1]
	// buf[2] is reserved
	addrType := buf[3]

	if version != SOCKS5_VERSION {
		return "", fmt.Errorf("unsupported SOCKS version: %d", version)
	}

	if cmd != CONNECT {
		return "", fmt.Errorf("unsupported command: %d", cmd)
	}

	var addr string
	switch addrType {
	case IPV4:
		ipBuf := make([]byte, 4)
		if _, err := io.ReadFull(conn, ipBuf); err != nil {
			return "", err
		}
		addr = net.IP(ipBuf).String()

	case DOMAIN:
		lenBuf := make([]byte, 1)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return "", err
		}
		domainLen := lenBuf[0]
		domainBuf := make([]byte, domainLen)
		if _, err := io.ReadFull(conn, domainBuf); err != nil {
			return "", err
		}
		addr = string(domainBuf)

	case IPV6:
		ipBuf := make([]byte, 16)
		if _, err := io.ReadFull(conn, ipBuf); err != nil {
			return "", err
		}
		addr = net.IP(ipBuf).String()

	default:
		return "", fmt.Errorf("unsupported address type: %d", addrType)
	}

	portBuf := make([]byte, 2)
	if _, err := io.ReadFull(conn, portBuf); err != nil {
		return "", err
	}
	port := binary.BigEndian.Uint16(portBuf)

	target := net.JoinHostPort(addr, strconv.Itoa(int(port)))
	return target, nil
}

func (p *SOCKS5Proxy) sendConnectResponse(conn net.Conn, status byte, bindAddr, bindPort string) error {
	response := []byte{
		SOCKS5_VERSION, // Version
		status,         // Status
		0x00,           // Reserved
		IPV4,           // Address type (IPv4)
	}

	ip := net.ParseIP(bindAddr).To4()
	if ip == nil {
		ip = []byte{0, 0, 0, 0}
	}
	response = append(response, ip...)

	port, _ := strconv.Atoi(bindPort)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, uint16(port))
	response = append(response, portBytes...)

	_, err := conn.Write(response)
	return err
}

func (p *SOCKS5Proxy) relay(conn1, conn2 net.Conn) {
	done := make(chan struct{}, 2)

	go func() {
		defer func() { done <- struct{}{} }()
		io.Copy(conn2, conn1)
	}()

	go func() {
		defer func() { done <- struct{}{} }()
		io.Copy(conn1, conn2)
	}()

	<-done
}

func main() {
	if len(os.Args) < 3 {
		log.Fatal("Usage: socks5-proxy <hostname> <auth-key> [port]")
	}

	hostname := os.Args[1]
	authkey := os.Args[2]
	port := "1080"
	if len(os.Args) > 3 {
		port = os.Args[3]
	}

	proxy := NewSOCKS5Proxy(hostname, authkey)
	
	log.Printf("Starting SOCKS5 proxy with hostname: %s", hostname)
	log.Printf("Connecting to Tailscale network...")
	log.Printf("Proxy will be available on port %s once connected", port)
	
	if err := proxy.Start(port); err != nil {
		log.Fatalf("Failed to start proxy: %v", err)
	}
}
