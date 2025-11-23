package mi_test

import (
	"bufio"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var cResponsePid = `SUCCESS: pid=10869
`

func TestReadResponsePid(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(cResponsePid))
	response, err := ReadResponse(reader)
	assert.Nil(t, err)
	pid, err := ParsePid(response)
	assert.Nil(t, err)
	assert.Equal(t, int64(10869), pid)
}

func TestReadResponseFailure(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(""))
	_, err := ReadResponse(reader)
	assert.NotNil(t, err)
}

func TestSendCommandFailure(t *testing.T) {
	conn := &net.IPConn{}
	err := SendCommand(conn, "dummy")
	assert.NotNil(t, err)
}
