package mi

import (
	"fmt"
	"net"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const statusOutput = `TITLE,OpenVPN 2.7.6 x86_64-pc-linux-gnu
TIME,Tue Aug 18 19:00:00 2026,1755543600
HEADER,CLIENT_LIST,Common Name,Real Address,Virtual Address,Virtual IPv6 Address,Bytes Received,Bytes Sent,Connected Since,Connected Since (time_t),Username,Client ID,Peer ID,Data Channel Cipher
CLIENT_LIST,laptop,203.0.113.7:51820,10.8.0.2,,10240,20480,Tue Aug 18 18:00:00 2026,1755540000,UNDEF,0,0,AES-256-GCM
CLIENT_LIST,phone,203.0.113.8:41234,10.8.0.3,fd00::3,512,1024,Tue Aug 18 18:30:00 2026,1755541800,UNDEF,1,1,CHACHA20-POLY1305
HEADER,ROUTING_TABLE,Virtual Address,Common Name,Real Address,Last Ref,Last Ref (time_t)
ROUTING_TABLE,10.8.0.2,laptop,203.0.113.7:51820,Tue Aug 18 19:00:00 2026,1755543600
GLOBAL_STATS,Max bcast/mcast queue length,0
END`

func TestParseStatus(t *testing.T) {
	connections := ParseStatus(statusOutput)
	require.Len(t, connections, 2)

	assert.Equal(t, "laptop", connections[0].Name)
	assert.Equal(t, "203.0.113.7:51820", connections[0].RealAddress)
	assert.Equal(t, "10.8.0.2", connections[0].VirtualIPv4)
	assert.Equal(t, "", connections[0].VirtualIPv6)
	assert.Equal(t, int64(10240), connections[0].BytesReceived)
	assert.Equal(t, int64(20480), connections[0].BytesSent)
	assert.Equal(t, time.Unix(1755540000, 0).UTC(), connections[0].ConnectedAt)
	assert.Equal(t, "AES-256-GCM", connections[0].Cipher)

	assert.Equal(t, "phone", connections[1].Name)
	assert.Equal(t, "fd00::3", connections[1].VirtualIPv6)
}

func TestParseStatus_NoClients(t *testing.T) {
	assert.Empty(t, ParseStatus("TITLE,OpenVPN 2.7.6\nEND"))
}

func TestParseStatus_ShortRowIgnored(t *testing.T) {
	assert.Empty(t, ParseStatus("CLIENT_LIST,truncated,1.2.3.4"))
}

func TestClient_ConnectionsOverSocket(t *testing.T) {
	socket := path.Join(t.TempDir(), "mgmt.sock")
	listener, err := net.Listen("unix", socket)
	require.NoError(t, err)
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		fmt.Fprintf(conn, ">INFO:OpenVPN Management Interface Version 5\n")
		buf := make([]byte, 128)
		if _, err := conn.Read(buf); err != nil {
			return
		}
		fmt.Fprintf(conn, "%s\n", statusOutput)
	}()

	client := &Client{Socket: socket}
	connections, err := client.Connections()
	require.NoError(t, err)
	require.Len(t, connections, 2)
	assert.Equal(t, "laptop", connections[0].Name)
}

func TestClient_SocketMissing(t *testing.T) {
	client := &Client{Socket: path.Join(t.TempDir(), "absent.sock")}
	_, err := client.Connections()
	assert.Error(t, err)
}

const versionOutput = `OpenVPN Version: OpenVPN 2.7.6 [git:modernize-2.7/a51ebac] x86_64-pc-linux-gnu [SSL (OpenSSL)] [LZO] [LZ4] [EPOLL] [MH/PKTINFO] [AEAD] built on Aug 19 2026
Management Version: 5
END`

func TestParseVersion(t *testing.T) {
	assert.Equal(t, "2.7.6", ParseVersion(versionOutput),
		"the UI shows this verbatim, so it must not be the whole build banner")
}

func TestParseVersion_Unparseable(t *testing.T) {
	assert.Equal(t, "", ParseVersion("Management Version: 5\nEND"))
}
