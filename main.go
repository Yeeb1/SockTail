package main

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

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
	DEFAULT_PORT   = "1080"
)

// Build-time injected obfuscated key (hex string)
var buildTimeObfuscatedKey string

// Build-time injected control server URL
var buildTimeControlURL string

var obfuscatedAuthKey = []byte{
	0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x20, 0x74, 0x68, 0x65, 0x72,
	0x65, 0x21, 0x20, 0x47, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x6c,
	0x20, 0x4b, 0x65, 0x6e, 0x6f, 0x62, 0x69, 0x2e,
}

var xorKey = []byte("747sg^8N0$")

type SOCKS5Proxy struct {
	server *tsnet.Server
}

func deobfuscateAuthKey() string {
	// If build-time key is available, use it
	if buildTimeObfuscatedKey != "" {
		if obfuscatedBytes, err := hex.DecodeString(buildTimeObfuscatedKey); err == nil {
			return string(xorDecode(obfuscatedBytes))
		}
	}
	// Fallback to embedded key
	return string(xorDecode(obfuscatedAuthKey))
}


func xorDecode(data []byte) []byte {
	result := make([]byte, len(data))
	for i, b := range data {
		result[i] = b ^ xorKey[i%len(xorKey)]
	}
	return result
}

func obfuscateAuthKey(key string) []byte {
	data := []byte(key)
	result := make([]byte, len(data))
	for i, b := range data {
		result[i] = b ^ xorKey[i%len(xorKey)]
	}
	return result
}

func obfuscateAuthKeyToHex(key string) string {
	return hex.EncodeToString(obfuscateAuthKey(key))
}

func generateHostname() string {
	// Generate a random hostname - a bit of entropy
	prefixes := []string{"web", "api", "cdn", "mail", "ftp", "db", "cache", "proxy", "gw", "vpn"}
	suffixes := []string{"srv", "node", "host", "box", "vm", "sys"}
	
	randBytes := make([]byte, 4)
	rand.Read(randBytes)
	
	prefixIdx := int(randBytes[0]) % len(prefixes)
	suffixIdx := int(randBytes[1]) % len(suffixes)
	num := int(randBytes[2])%100 + 1
	
	return fmt.Sprintf("%s-%s-%02d", prefixes[prefixIdx], suffixes[suffixIdx], num)
}

func getSystemHostname() string {
	if hostname, err := os.Hostname(); err == nil {
		hostname = strings.Split(hostname, ".")[0]
		hostname = strings.ReplaceAll(hostname, "_", "-")
		if len(hostname) > 0 && len(hostname) <= 63 {
			return hostname
		}
	}
	
	return generateHostname()
}

func NewSOCKS5Proxy(hostname, authkey, controlURL string) *SOCKS5Proxy {
	if hostname == "" {
		hostname = getSystemHostname()
	}

	if authkey == "" {
		authkey = deobfuscateAuthKey()
	}

	if controlURL == "" && buildTimeControlURL != "" {
		controlURL = buildTimeControlURL
	}

	s := &tsnet.Server{
		Hostname:   hostname,
		AuthKey:    authkey,
		ControlURL: controlURL,
		Logf:       func(format string, args ...interface{}) {},
	}
	return &SOCKS5Proxy{server: s}
}

func (p *SOCKS5Proxy) Start(port string) error {
	listener, err := p.server.Listen("tcp", ":"+port)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}
	defer listener.Close()

	log.Printf("SOCKS5 proxy active on %s:%s", p.server.Hostname, port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go p.handleConnection(conn)
	}
}

func (p *SOCKS5Proxy) handleConnection(conn net.Conn) {
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(30 * time.Second))

	// Step 1: Handle authentication negotiation
	if err := p.handleAuth(conn); err != nil {
		return
	}

	// Step 2: Handle CONNECT request
	target, err := p.handleConnect(conn)
	if err != nil {
		return
	}

	// Step 3: Establish connection to target through Tailscale
	targetConn, err := p.server.Dial(context.Background(), "tcp", target)
	if err != nil {
		p.sendConnectResponse(conn, FAILURE, "0.0.0.0", "0")
		return
	}
	defer targetConn.Close()

	if err := p.sendConnectResponse(conn, SUCCESS, "0.0.0.0", "0"); err != nil {
		return
	}

	conn.SetDeadline(time.Time{})
	targetConn.SetDeadline(time.Time{})

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

	// Add bind address (4 bytes for IPv4)
	ip := net.ParseIP(bindAddr).To4()
	if ip == nil {
		ip = []byte{0, 0, 0, 0}
	}
	response = append(response, ip...)

	// Add bind port (2 bytes)
	port, _ := strconv.Atoi(bindPort)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, uint16(port))
	response = append(response, portBytes...)

	_, err := conn.Write(response)
	return err
}

func (p *SOCKS5Proxy) relay(conn1, conn2 net.Conn) {
	done := make(chan struct{}, 2)

	// Copy data from conn1 to conn2
	go func() {
		defer func() { done <- struct{}{} }()
		io.Copy(conn2, conn1)
	}()

	// Copy data from conn2 to conn1
	go func() {
		defer func() { done <- struct{}{} }()
		io.Copy(conn1, conn2)
	}()

	// Wait for one direction to complete
	<-done
}

func printUsage() {
	fmt.Printf("Usage: %s [hostname] [authkey] [control-url]\n", os.Args[0])
	fmt.Println("  hostname:    Optional. Auto-generated if not specified")
	fmt.Println("  authkey:     Optional. Uses embedded key if not specified")
	fmt.Println("  control-url: Optional. Uses Tailscale default or build-time URL if not specified")
	fmt.Println("  port:        Fixed at 1080")
	fmt.Println("\nExamples:")
	fmt.Printf("  %s                                    # Auto hostname, embedded auth, default control\n", os.Args[0])
	fmt.Printf("  %s my-proxy                           # Custom hostname, embedded auth, default control\n", os.Args[0])
	fmt.Printf("  %s my-proxy tskey-auth-...            # Custom hostname and auth, default control\n", os.Args[0])
	fmt.Printf("  %s my-proxy tskey-auth-... https://headscale.example.com  # All custom\n", os.Args[0])
}

func main() {
	var hostname, authkey, controlURL string

	switch len(os.Args) {
	case 1:
	case 2:
		if os.Args[1] == "-h" || os.Args[1] == "--help" {
			printUsage()
			return
		}
		hostname = os.Args[1]
	case 3:
		hostname = os.Args[1]
		authkey = os.Args[2]
	case 4:
		hostname = os.Args[1]
		authkey = os.Args[2]
		controlURL = os.Args[3]
	default:
		printUsage()
		os.Exit(1)
	}

	proxy := NewSOCKS5Proxy(hostname, authkey, controlURL)

	if hostname == "" {
		hostname = getSystemHostname()
	}

	controlServer := "default Tailscale"
	if controlURL != "" {
		controlServer = controlURL
	} else if buildTimeControlURL != "" {
		controlServer = buildTimeControlURL
	}

	log.Printf("Starting proxy as %s", hostname)
	log.Printf("Using control server: %s", controlServer)

	if err := proxy.Start(DEFAULT_PORT); err != nil {
		log.Fatalf("Proxy failed: %v", err)
	}
}
