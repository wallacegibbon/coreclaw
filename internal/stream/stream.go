package stream

import (
	"encoding/binary"
	"io"
)

// Message tags for TLV protocol
const (
	TagText      = 'T' // Regular text output
	TagTool      = 't' // Tool call output
	TagReasoning = 'R' // Reasoning/thinking content
	TagError     = 'E' // Error messages
)

// WriteTLV writes a TLV message to the output
func WriteTLV(output Output, tag byte, value string) error {
	data := []byte(value)
	length := int32(len(data))

	// Build complete message: tag (1) + length (4) + value
	msg := make([]byte, 5+length)
	msg[0] = tag
	binary.BigEndian.PutUint32(msg[1:], uint32(length))
	copy(msg[5:], data)

	// Write complete message in one call
	_, err := output.Write(msg)
	return err
}

// ReadTLV reads a TLV message from the input
// Returns tag, value, and error
func ReadTLV(input Input) (byte, string, error) {
	// Read tag (1 byte)
	tagBuf := make([]byte, 1)
	_, err := input.Read(tagBuf)
	if err != nil {
		return 0, "", err
	}
	tag := tagBuf[0]

	// Read length (4 bytes)
	lenBuf := make([]byte, 4)
	_, err = input.Read(lenBuf)
	if err != nil {
		return 0, "", err
	}
	length := binary.BigEndian.Uint32(lenBuf)

	// Read value
	valueBuf := make([]byte, length)
	_, err = input.Read(valueBuf)
	if err != nil {
		return 0, "", err
	}

	return tag, string(valueBuf), nil
}

// Input defines the input interface for the agent processor
type Input interface {
	Read(p []byte) (n int, err error)
}

// Output defines the output interface for the agent processor
type Output interface {
	Write(p []byte) (n int, err error)
	WriteString(s string) (n int, err error)
	Flush() error
}

// ReadCloser combines Input with io.Closer
type ReadCloser struct {
	Input
}

func (rc *ReadCloser) Close() error {
	return nil
}

// WriteCloser combines Output with io.Closer
type WriteCloser struct {
	Output
}

func (wc *WriteCloser) Close() error {
	return nil
}

// NopInput is an Input that returns EOF
type NopInput struct{}

func (n *NopInput) Read(p []byte) (int, error) {
	return 0, io.EOF
}

// NopOutput is an Output that discards all output
type NopOutput struct{}

func (n *NopOutput) Write(p []byte) (int, error) {
	return len(p), nil
}

func (n *NopOutput) WriteString(s string) (int, error) {
	return len(s), nil
}

func (n *NopOutput) Flush() error {
	return nil
}
