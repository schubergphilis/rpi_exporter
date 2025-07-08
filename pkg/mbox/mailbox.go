package mbox

import (
	"errors"
	"fmt"
	"os"
	"unsafe"

	"github.com/schubergphilis/rpi_exporter/pkg/ioctl"
	log "github.com/sirupsen/logrus"
)

const (
	RequestCodeDefault        = 0x00000000
	MailboxBufferAlignment    = 16
	MailboxDefaultBufferWords = 48
	MailboxWordBytes          = 4
	MailboxHeaderWords        = 2 // m.buf[0] and m.buf[1]: size and code
	MailboxTagFields          = 3 // id, valueBufSize, request/resp code
	MailboxRequestHeaderWords = 5 // size, code, tag id, valueBufSize, request/resp
	MailboxRequestHeaderBytes = MailboxRequestHeaderWords * MailboxWordBytes
	MailboxDebugPrintWordsTX  = 5  // header + tag
	MailboxDebugPrintWordsRX  = 16 // for debug dump (arbitrary, safe for most responses)
	MailboxEndTagValue        = 0
	MailboxEndTagWords        = 1
	MailboxMinCompleteTagLen  = 3 // id, valuebufsize, len/resp field
	PowerStateReturnIdx       = 1
	ClockRateReturnIdx        = 1
	GetUint32ReturnIdx        = 0
	PowerStateMask            = 0x03
	MailboxResponseLenMask    = 0x7FFFFFFF
	MailboxResponseSuccessBit = 0x80000000
	MailboxMilliScale         = 1000
	MailboxMicroScale         = 1000000
	MailboxTwoWords           = 2
)

const (
	TagGetFirmwareRevision  = 0x00000001
	TagGetBoardModel        = 0x00010001
	TagGetBoardRevision     = 0x00010002
	TagGetBoardMAC          = 0x00010003
	TagGetPowerState        = 0x00020001
	TagGetClockRate         = 0x00030002
	TagGetVoltage           = 0x00030003
	TagGetMaxVoltage        = 0x00030005
	TagGetTemperature       = 0x00030006
	TagGetMinVoltage        = 0x00030008
	TagGetTurbo             = 0x00030009
	TagGetMaxTemperature    = 0x0003000A
	TagGetClockRateMeasured = 0x00030047
)

const (
	replySuccess = 0x80000000
	replyFail    = 0x80000001
)

var Debug = false

var (
	ErrNotImplemented = errors.New("vcio: not implemented")
	ErrRequestBuffer  = errors.New("vcio: error parsing request buffer")
)

var mbIoctl = ioctl.IOWR('d', 0, uint(unsafe.Sizeof(new(byte))))

type Tag []uint32

var EndTag = Tag{MailboxEndTagValue}

func (t Tag) ID() uint32 {
	if !t.IsValid() {
		return 0
	}

	return t[0]
}

// Cap returns the length of the value buffer in bytes.
func (t Tag) Cap() int {
	if !t.IsValid() {
		return 0
	}

	return int(t[1])
}

// Len returns the length of a response value in bytes.
func (t Tag) Len() int {
	if !t.IsValid() || !t.IsResponse() {
		return 0
	}

	return int(t[2] & MailboxResponseLenMask)
}

func (t Tag) IsResponse() bool {
	if !t.IsValid() {
		return false
	}

	return t[2]&MailboxResponseSuccessBit == MailboxResponseSuccessBit
}

// Value returns the value buffer. TODO: Always 32bit.
func (t Tag) Value() []uint32 {
	if !t.IsValid() {
		return nil
	}

	return t[3 : 3+t.Len()/MailboxWordBytes]
}

func (t Tag) IsEnd() bool {
	return len(t) == MailboxEndTagWords && t[0] == MailboxEndTagValue
}

func (t Tag) IsValid() bool {
	if len(t) == 0 {
		return false // Nil or empty
	}

	if len(t) == MailboxEndTagWords {
		return t.IsEnd() // End tag
	}

	if len(t) < MailboxMinCompleteTagLen {
		return false // Too short
	}

	if len(t) != int(MailboxMinCompleteTagLen+t[1]/MailboxWordBytes) {
		return false // Incorrect size with value buffer
	}

	return true
}

func ReadTag(tag []uint32) (Tag, error) {
	if len(tag) > 0 && tag[0] == MailboxEndTagValue {
		return EndTag, nil
	}

	if len(tag) < MailboxMinCompleteTagLen {
		return nil, errors.New("vcio: tag buffer is too small")
	}

	sz := MailboxMinCompleteTagLen + int(tag[1]/MailboxWordBytes)
	if len(tag) < sz {
		return nil, errors.New("vcio: tag buffer is too small")
	}

	return Tag(tag[:sz]), nil
}

// Mailbox implements the Mailbox protocol used by the VideoCore and ARM on a Raspberry Pi.
type Mailbox struct {
	f            *os.File
	bufUnaligned [MailboxDefaultBufferWords]uint32
	buf          []uint32
}

func Open() (*Mailbox, error) {
	vcioFile, err := os.OpenFile("/dev/vcio", os.O_RDONLY, os.ModePerm)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotImplemented
	}

	if err != nil {
		return nil, fmt.Errorf("unable to open vcioFile: %w", err)
	}

	return &Mailbox{f: vcioFile}, nil
}

func (m *Mailbox) Close() {
	if m == nil || m.f == nil {
		return
	}

	if err := m.f.Close(); err != nil {
		log.WithError(err).Error("unable to close mail box")
	}

	m.f = nil
}

// Do sends a single command tag and returns all response tags. Returned memory is only usable until
// the next request is made.
func (m *Mailbox) Do(tagID uint32, bufferBytes int, args ...uint32) ([]Tag, error) {
	m.alignBuffer()
	bufferBytes = m.ensureBufferSize(bufferBytes, len(args))
	m.writeRequestHeader(bufferBytes, tagID, args)

	debugf("TX:\n")
	m.debugBuffer("  %02d: 0x%08X\n", m.buf[:MailboxRequestHeaderWords+len(args)])

	if err := m.sendIOCTL(); err != nil {
		return nil, fmt.Errorf("unable to send message via ioctl: %w", err)
	}

	debugf("RX:\n")
	m.debugBuffer("  %02d: 0x%08X\n", m.buf[:MailboxDebugPrintWordsRX])

	if err := m.checkResponse(); err != nil {
		return nil, err
	}

	return m.readAllTags()
}

// GetFirmwareRevision returns the firmware revision of the VideoCore component.
func (m *Mailbox) GetFirmwareRevision() (uint32, error) {
	return m.getUint32(TagGetFirmwareRevision)
}

// GetBoardModel returns the model number of the system board.
func (m *Mailbox) GetBoardModel() (uint32, error) {
	return m.getUint32(TagGetBoardModel)
}

// GetBoardRevision returns the revision number of the system board.
func (m *Mailbox) GetBoardRevision() (uint32, error) {
	return m.getUint32(TagGetBoardRevision)
}

// PowerDeviceID identifiers.
type PowerDeviceID uint32

const (
	PowerDeviceIDSDCard PowerDeviceID = 0x00000000
	PowerDeviceIDUART0  PowerDeviceID = 0x00000001
	PowerDeviceIDUART1  PowerDeviceID = 0x00000002
	PowerDeviceIDUSBHCD PowerDeviceID = 0x00000003
	PowerDeviceIDI2C0   PowerDeviceID = 0x00000004
	PowerDeviceIDI2C1   PowerDeviceID = 0x00000005
	PowerDeviceIDI2C2   PowerDeviceID = 0x00000006
	PowerDeviceIDSPI    PowerDeviceID = 0x00000007
	PowerDeviceIDCCP2TX PowerDeviceID = 0x00000008
	// PowerDeviceIDUnknown (RPi4) PowerDeviceID = 0x00000009, 0x0000000a.
)

type PowerState uint32

const (
	PowerStateOn      uint32 = 0x00000001
	PowerStateOff     uint32 = 0x00000001
	PowerStateMissing uint32 = 0x00000010
)

func (m *Mailbox) GetPowerState(id PowerDeviceID) (PowerState, error) {
	tags, err := m.Do(
		TagGetPowerState,
		MailboxTwoWords*MailboxWordBytes, uint32(id),
	)
	if err != nil {
		return 0, err
	}

	return PowerState(tags[0].Value()[PowerStateReturnIdx] & PowerStateMask), nil
}

// ClockID identifies a clock.
type ClockID uint32

const (
	ClockIDEMMC     ClockID = 0x00000001
	ClockIDUART     ClockID = 0x00000002
	ClockIDARM      ClockID = 0x00000003
	ClockIDCore     ClockID = 0x00000004
	ClockIDV3D      ClockID = 0x00000005
	ClockIDH264     ClockID = 0x00000006
	ClockIDISP      ClockID = 0x00000007
	ClockIDSDRAM    ClockID = 0x00000008
	ClockIDPixel    ClockID = 0x00000009
	ClockIDPWM      ClockID = 0x0000000a
	ClockIDHEVC     ClockID = 0x0000000b
	ClockIDEMMC2    ClockID = 0x0000000c
	ClockIDM2MC     ClockID = 0x0000000d
	ClockIDPixelBVB ClockID = 0x0000000e
)

func (m *Mailbox) GetClockRate(id ClockID) (int, error) {
	v, err := m.getUint32ByID(TagGetClockRate, uint32(id))
	if err != nil {
		return 0, err
	}

	return int(v), nil
}

func (m *Mailbox) GetClockRateMeasured(id ClockID) (int, error) {
	v, err := m.getUint32ByID(TagGetClockRateMeasured, uint32(id))
	if err != nil {
		return 0, err
	}

	return int(v), nil
}

// GetTemperature returns the temperature of the SoC in degrees celsius.
func (m *Mailbox) GetTemperature() (float32, error) {
	return m.getTemperature(TagGetTemperature)
}

// GetMaxTemperature returns the maximum safe temperature of the SoC in degrees celsius.
// Overclock may be disabled above this temperature.
func (m *Mailbox) GetMaxTemperature() (float32, error) {
	return m.getTemperature(TagGetMaxTemperature)
}

// VoltageID identifies a voltage rail.
type VoltageID uint32

const (
	VoltageIDCore   VoltageID = 0x00000001
	VoltageIDSDRAMC VoltageID = 0x00000002
	VoltageIDSDRAMP VoltageID = 0x00000003
	VoltageIDSDRAMI VoltageID = 0x00000004
)

// GetVoltage returns the voltage of the given component.
func (m *Mailbox) GetVoltage(id VoltageID) (float32, error) {
	return m.getVoltage(TagGetVoltage, id)
}

// GetMinVoltage returns the minimum supported voltage of the given component.
func (m *Mailbox) GetMinVoltage(id VoltageID) (float32, error) {
	return m.getVoltage(TagGetMinVoltage, id)
}

// GetMaxVoltage returns the maximum supported voltage of the given component.
func (m *Mailbox) GetMaxVoltage(id VoltageID) (float32, error) {
	return m.getVoltage(TagGetMaxVoltage, id)
}

func (m *Mailbox) GetTurbo() (bool, error) {
	tags, err := m.Do(
		TagGetTurbo,
		MailboxTwoWords*MailboxWordBytes, 0,
	)
	if err != nil {
		return false, err
	}

	return tags[0].Value()[PowerStateReturnIdx] == 1, nil
}

// alignBuffer ensures the buffer is aligned to a 16-byte boundary.
func (m *Mailbox) alignBuffer() {
	if m.buf == nil {
		offset := uintptr(unsafe.Pointer(&m.bufUnaligned[0])) & (MailboxBufferAlignment - 1)
		m.buf = m.bufUnaligned[MailboxBufferAlignment-offset : uintptr(len(m.bufUnaligned))-offset]
	}
}

// ensureBufferSize sets the buffer size ensuring it can hold all args.
func (m *Mailbox) ensureBufferSize(bufferBytes, numArgs int) int {
	minBytes := numArgs * MailboxWordBytes
	if bufferBytes < minBytes {
		return minBytes
	}

	return bufferBytes
}

// writeRequestHeader writes the message header and tag into the buffer.
func (m *Mailbox) writeRequestHeader(bufferBytes int, tagID uint32, args []uint32) {
	m.buf[0] = uint32(len(m.buf)) * MailboxWordBytes
	m.buf[1] = RequestCodeDefault
	m.buf[2] = tagID
	m.buf[3] = uint32(bufferBytes)
	m.buf[4] = 0 // request
	copy(m.buf[MailboxRequestHeaderWords:], args)
}

// debugBuffer prints out buffer values for debugging.
func (m *Mailbox) debugBuffer(format string, buf []uint32) {
	for i, v := range buf {
		debugf(format, i, v)
	}
}

// sendIOCTL sends the buffer via ioctl.
func (m *Mailbox) sendIOCTL() error {
	if err := ioctl.Ioctl(m.f.Fd(), uintptr(mbIoctl), uintptr(unsafe.Pointer(&m.buf[0]))); err != nil {
		return fmt.Errorf("failed to send via ioctl: %w", err)
	}

	return nil
}

// checkResponse checks for errors in the response header.
func (m *Mailbox) checkResponse() error {
	switch {
	case m.buf[1] == replyFail:
		return ErrRequestBuffer
	case m.buf[1]&replySuccess != replySuccess:
		return fmt.Errorf("vcio: unexpected response code: 0x%08x", m.buf[1])
	}

	return nil
}

// readAllTags parses all tags from response buffer.
func (m *Mailbox) readAllTags() ([]Tag, error) {
	remaining := m.buf[2:]

	var tags []Tag

	for {
		tag, err := ReadTag(remaining)
		if err != nil {
			return nil, err
		}

		if tag.IsEnd() {
			break
		}

		tags = append(tags, tag)
		remaining = remaining[len(tag):]
	}

	return tags, nil
}

func (m *Mailbox) getUint32(tagID uint32) (uint32, error) {
	tags, err := m.Do(tagID, MailboxWordBytes)
	if err != nil {
		return 0, err
	}

	return tags[0].Value()[GetUint32ReturnIdx], nil
}

func (m *Mailbox) getUint32ByID(tagID, id uint32) (uint32, error) {
	tags, err := m.Do(tagID, MailboxTwoWords*MailboxWordBytes, id)
	if err != nil {
		return 0, err
	}

	return tags[0].Value()[ClockRateReturnIdx], nil
}

func debugf(format string, a ...interface{}) {
	if !Debug {
		return
	}

	fmt.Fprintf(os.Stderr, format, a...)
}

func (m *Mailbox) getTemperature(tag uint32) (float32, error) {
	v, err := m.getUint32ByID(tag, 0)
	if err != nil {
		return 0, err
	}

	return float32(v) / MailboxMilliScale, nil
}

func (m *Mailbox) getVoltage(tag uint32, id VoltageID) (float32, error) {
	v, err := m.getUint32ByID(tag, uint32(id))
	if err != nil {
		return 0, err
	}

	return float32(v) / MailboxMicroScale, nil
}
