package danmaku

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"crypto/tls"

	"github.com/gorilla/websocket"
)

const (
	wsURL             = "wss://danmuproxy.douyu.com:8506/"
	heartbeatInterval = 30 * time.Second
)

type Client struct {
	roomURL string
	prefix  string
	onCmd   func(string)
}

func New(roomURL, prefix string, onCmd func(string)) *Client {
	return &Client{
		roomURL: roomURL,
		prefix:  prefix,
		onCmd:   onCmd,
	}
}

func (c *Client) Start(ctx context.Context) error {
	roomID, err := resolveRoomID(ctx, c.roomURL)
	if err != nil {
		return err
	}
	log.Printf("danmaku: resolved room id=%s from %s", roomID, c.roomURL)

	dialer := *websocket.DefaultDialer
	dialer.TLSClientConfig = &tls.Config{
		ServerName: "danmuproxy.douyu.com",
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		},
	}
	dialer.HandshakeTimeout = 15 * time.Second

	headers := http.Header{}
	headers.Set("Origin", "https://www.douyu.com")
	headers.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36")
	headers.Set("Pragma", "no-cache")
	headers.Set("Cache-Control", "no-cache")

	conn, resp, err := dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		if resp != nil {
			log.Printf("danmaku: websocket handshake response status=%s headers=%v", resp.Status, resp.Header)
		}
		return err
	}
	log.Printf("danmaku: connected websocket %s", wsURL)

	if err := writePacket(conn, fmt.Sprintf("type@=loginreq/roomid@=%s/", roomID)); err != nil {
		_ = conn.Close()
		return err
	}
	if err := writePacket(conn, fmt.Sprintf("type@=joingroup/rid@=%s/gid@=-9999/", roomID)); err != nil {
		_ = conn.Close()
		return err
	}

	go c.heartbeat(ctx, conn)
	go c.readLoop(ctx, conn)
	return nil
}

func (c *Client) heartbeat(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()
	defer conn.Close()

	for {
		select {
		case <-ctx.Done():
			log.Printf("danmaku: heartbeat loop stopped")
			return
		case <-ticker.C:
			_ = writeRaw(conn, []byte{0x14, 0x00, 0x00, 0x00, 0x14, 0x00, 0x00, 0x00, 0xb1, 0x02, 0x00, 0x00, 0x74, 0x79, 0x70, 0x65, 0x40, 0x3d, 0x6d, 0x72, 0x6b, 0x6c, 0x2f, 0x00})
		}
	}
}

func (c *Client) readLoop(ctx context.Context, conn *websocket.Conn) {
	defer conn.Close()

	for {
		select {
		case <-ctx.Done():
			log.Printf("danmaku: read loop stopped")
			return
		default:
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("danmaku: read message failed: %v", err)
			return
		}

		for _, msg := range decodeMessages(data) {
			if strings.HasPrefix(msg, c.prefix) {
				log.Printf("danmaku: command received: %s", strings.TrimSpace(msg))
				c.onCmd(strings.TrimSpace(msg))
			}
		}
	}
}

func resolveRoomID(ctx context.Context, roomURL string) (string, error) {
	if fromURL := roomIDFromURL(roomURL); fromURL != "" {
		return fromURL, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, roomURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	match := regexp.MustCompile(`\$ROOM\.room_id\s*=\s*(\d+)`).FindStringSubmatch(string(body))
	if len(match) != 2 {
		return "", fmt.Errorf("failed to resolve douyu room id from %s", roomURL)
	}
	return match[1], nil
}

func roomIDFromURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	segment := strings.Trim(strings.TrimSpace(parsed.Path), "/")
	if segment == "" {
		return ""
	}
	if _, err := strconv.Atoi(segment); err == nil {
		return segment
	}
	return ""
}

func writePacket(conn *websocket.Conn, payload string) error {
	data := append([]byte(payload), 0x00)
	size := int32(len(data) + 8)
	packet := make([]byte, 0, len(data)+12)
	lenBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBuf, uint32(size))
	packet = append(packet, lenBuf...)
	packet = append(packet, lenBuf...)
	packet = append(packet, 0xb1, 0x02, 0x00, 0x00)
	packet = append(packet, data...)
	return writeRaw(conn, packet)
}

func writeRaw(conn *websocket.Conn, data []byte) error {
	return conn.WriteMessage(websocket.BinaryMessage, data)
}

func decodeMessages(data []byte) []string {
	raw := string(data)
	var out []string
	for _, part := range strings.Split(raw, "\x00") {
		if !strings.Contains(part, "type@=") {
			continue
		}

		msgType := extractField(part, "type")
		if msgType != "chatmsg" {
			continue
		}
		if text := extractField(part, "txt"); text != "" {
			out = append(out, decodeEscapes(text))
		}
	}
	return out
}

func extractField(body, key string) string {
	prefix := key + "@="
	idx := strings.Index(body, prefix)
	if idx < 0 {
		return ""
	}
	rest := body[idx+len(prefix):]
	end := strings.Index(rest, "/")
	if end < 0 {
		return rest
	}
	return rest[:end]
}

func decodeEscapes(v string) string {
	v = strings.ReplaceAll(v, "@S", "/")
	v = strings.ReplaceAll(v, "@A", "@")
	return v
}

func ParseIndexCommand(cmd, prefix string) (int, bool) {
	if !strings.HasPrefix(cmd, prefix) {
		return 0, false
	}
	raw := strings.TrimSpace(strings.TrimPrefix(cmd, prefix))
	if raw == "" {
		return 0, false
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}
