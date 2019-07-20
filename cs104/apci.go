package cs104

import (
	"fmt"

	"github.com/thinkgos/go-iecp5/asdu"
)

const (
	startFrame byte = 0x68 // 起动字符
)

// APDU form  Max size 255
//      |              APCI                   |       ASDU         |
//      | start | APDU length | control field |       ASDU         |
//                       |          APDU field size(253)           |
// bytes|    1  |    1   |        4           |                    |
const (
	APCICtlFiledSize = 4 // control filed(4)

	APDUSizeMax      = 255                                 // start(1) + length(1) + control field(4) + ASDU
	APDUFieldSizeMax = APCICtlFiledSize + asdu.ASDUSizeMax // control field(4) + ASDU
)

const (
	uFrame = "U" // U帧 只含apci 未编号控制信息 unnumbered
	sFrame = "S" // S帧 只含apci S帧用于主要用确认帧的正确传输,协议称是监视. supervisory
	iFrame = "I" // i帧 含apci和asdu 信息帧. information
)

// U帧 控制域功能
const (
	uStartDtActive  = 4 << iota // 启动激活 0x04
	uStartDtConfirm             // 启动确认 0x08
	uStopDtActive               // 停止激活 0x10
	uStopDtConfirm              // 停止确认 0x20
	uTestFrActive               // 测试激活 0x40
	uTestFrConfirm              // 测试确认 0x80
)

// APCI apci 应用规约控制信息
type APCI struct {
	start                  byte
	apduFiledLen           byte // control + asdu 的长度
	ctr1, ctr2, ctr3, ctr4 byte
}

// I格式 用于编号的信息传输
type iAPCI struct {
	sendSN, rcvSN uint16
}

// S格式 编号的监视功能
type sAPCI struct {
	rcvSN uint16
}

// U格式 未编号的控制功能
type uAPCI struct {
	function byte // bit8 测试确认
}

// 将 apci 解析到I,S,U帧
func (this APCI) parse() (interface{}, string) {
	if this.ctr1&0x01 == 0 {
		return iAPCI{
			sendSN: uint16(this.ctr1)>>1 + uint16(this.ctr2)<<7,
			rcvSN:  uint16(this.ctr3)>>1 + uint16(this.ctr4)<<7,
		}, iFrame
	}

	if this.ctr1&0x03 == 0x01 {
		return sAPCI{
			rcvSN: uint16(this.ctr3)>>1 + uint16(this.ctr4)<<7,
		}, sFrame
	}

	// this.ctrl&0x03 == 0x03
	return uAPCI{
		function: this.ctr1 & 0xfc,
	}, uFrame
}

// String 返回apci的帧格式
func (this APCI) String() string {
	acpi, format := this.parse()

	switch format {
	case sFrame:
		return fmt.Sprintf("S[recvNo: %d]", acpi.(sAPCI).rcvSN)
	case iFrame:
		return fmt.Sprintf("I[sendNo: %d, recvNo: %d]", acpi.(iAPCI).rcvSN, acpi.(iAPCI).sendSN)
	default: // uFrame
		var s string
		switch acpi.(uAPCI).function {
		case uStartDtActive: // 启动激活 0x04
			s = "StartDtActive"
		case uStartDtConfirm: // 启动确认 0x08
			s = "StartDtConfirm"
		case uStopDtActive: // 停止激活 0x10
			s = "StopDtActive"
		case uStopDtConfirm: // 停止确认 0x20
			s = "StopDtConfirm"
		case uTestFrActive: // 测试激活 0x40
			s = "TestFrActive"
		case uTestFrConfirm: // 测试确认 0x80
			s = "TestFrConfirm"
		default:
			s = "Unknown"
		}
		return fmt.Sprintf("U[function: %s]", s)
	}
}

// newIFrame 创建I帧 ,返回apdu
func newIFrame(asdus []byte, sendSN, RcvSN uint16) ([]byte, error) {
	if len(asdus) > asdu.ASDUSizeMax {
		return nil, fmt.Errorf("ASDU filed large than max %d", asdu.ASDUSizeMax)
	}

	b := make([]byte, len(asdus)+6)

	b[0] = startFrame
	b[1] = byte(len(asdus) + 4)
	b[2] = byte(sendSN << 1)
	b[3] = byte(sendSN >> 7)
	b[4] = byte(RcvSN << 1)
	b[5] = byte(RcvSN >> 7)
	copy(b[6:], asdus)

	return b, nil
}

// newSFrame 创建S帧,返回apdu
func newSFrame(RcvSN uint16) []byte {
	return []byte{startFrame, 4, 0x01, 0x00, byte(RcvSN << 1), byte(RcvSN >> 7)}
}

// newUFrame 创建U帧,返回apdu
func newUFrame(which int) []byte {
	return []byte{startFrame, 4, byte(which | 0x03), 0x00, 0x00, 0x00}
}

// return
func parse(apdu []byte) (APCI, []byte) {
	return APCI{apdu[0], apdu[1], apdu[2], apdu[3], apdu[4], apdu[5]}, apdu[6:]
}
