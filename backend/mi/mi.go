package mi

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

const dialTimeout = 5 * time.Second

type Connection struct {
	Name          string    `json:"name"`
	RealAddress   string    `json:"real_address"`
	VirtualIPv4   string    `json:"virtual_ipv4"`
	VirtualIPv6   string    `json:"virtual_ipv6"`
	BytesReceived int64     `json:"bytes_received"`
	BytesSent     int64     `json:"bytes_sent"`
	ConnectedAt   time.Time `json:"connected_at"`
	Cipher        string    `json:"cipher"`
}

type Client struct {
	Socket string
}

func (c *Client) Connections() ([]Connection, error) {
	out, err := c.execute("status 2")
	if err != nil {
		return nil, err
	}
	return ParseStatus(out), nil
}

func (c *Client) Version() (string, error) {
	out, err := c.execute("version")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(out, "\n") {
		if version, found := strings.CutPrefix(strings.TrimSpace(line), "OpenVPN Version: "); found {
			return version, nil
		}
	}
	return "", nil
}

func (c *Client) execute(command string) (string, error) {
	conn, err := net.DialTimeout("unix", c.Socket, dialTimeout)
	if err != nil {
		return "", fmt.Errorf("dial %s: %w", c.Socket, err)
	}
	defer conn.Close()

	if err := conn.SetDeadline(time.Now().Add(dialTimeout)); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintf(conn, "%s\n", command); err != nil {
		return "", err
	}

	var builder strings.Builder
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, ">") {
			continue
		}
		if trimmed == "END" || strings.HasPrefix(trimmed, "ERROR:") {
			break
		}
		builder.WriteString(line)
		builder.WriteString("\n")
		if strings.HasPrefix(trimmed, "SUCCESS:") {
			break
		}
	}
	return builder.String(), scanner.Err()
}

func ParseStatus(output string) []Connection {
	connections := []Connection{}
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Split(strings.TrimRight(line, "\r"), ",")
		if len(fields) < 10 || fields[0] != "CLIENT_LIST" {
			continue
		}
		connections = append(connections, Connection{
			Name:          fields[1],
			RealAddress:   fields[2],
			VirtualIPv4:   fields[3],
			VirtualIPv6:   fields[4],
			BytesReceived: parseInt(fields[5]),
			BytesSent:     parseInt(fields[6]),
			ConnectedAt:   parseUnix(fields[8]),
			Cipher:        cipher(fields),
		})
	}
	return connections
}

func cipher(fields []string) string {
	if len(fields) < 13 {
		return ""
	}
	return fields[12]
}

func parseInt(value string) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func parseUnix(value string) time.Time {
	seconds := parseInt(value)
	if seconds == 0 {
		return time.Time{}
	}
	return time.Unix(seconds, 0).UTC()
}
