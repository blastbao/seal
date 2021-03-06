package co

import (
	"encoding/binary"
	"log"
	"seal/rtmp/pt"

	"github.com/calabashdad/utiltools"
)

func (rc *RtmpConn) sendMsg(msg *pt.Message) (err error) {
	defer func() {
		if err := recover(); err != nil {
			log.Println(utiltools.PanicTrace())
		}
	}()

	if nil == msg {
		return
	}

	// ensure the basic header is 1bytes. make simple.
	if msg.Header.PerferCsid < 2 {
		msg.Header.PerferCsid = pt.RtmpCidProtocolControl
	}

	// current position of payload send.
	var payloadOffset uint32

	// always write the header event payload is Empty.
	for {
		if payloadOffset >= msg.Header.PayloadLength {
			break
		}

		var headerOffset uint32
		var header [pt.RtmpMaxFmt0HeaderSize]uint8

		if 0 == payloadOffset {
			// write new chunk stream header, fmt is 0
			header[headerOffset] = 0x00 | uint8(msg.Header.PerferCsid&0x3f)
			headerOffset++

			// chunk message header, 11 bytes
			// timestamp, 3bytes, big-endian
			timestamp := msg.Header.Timestamp
			if timestamp < pt.RtmpExtendTimeStamp {
				header[headerOffset] = uint8((timestamp & 0x00ff0000) >> 16)
				headerOffset++
				header[headerOffset] = uint8((timestamp & 0x0000ff00) >> 8)
				headerOffset++
				header[headerOffset] = uint8(timestamp & 0x000000ff)
				headerOffset++
			} else {
				header[headerOffset] = 0xff
				headerOffset++
				header[headerOffset] = 0xff
				headerOffset++
				header[headerOffset] = 0xff
				headerOffset++
			}

			// message_length, 3bytes, big-endian
			payloadLengh := msg.Header.PayloadLength
			header[headerOffset] = uint8((payloadLengh & 0x00ff0000) >> 16)
			headerOffset++
			header[headerOffset] = uint8((payloadLengh & 0x0000ff00) >> 8)
			headerOffset++
			header[headerOffset] = uint8((payloadLengh & 0x000000ff))
			headerOffset++

			// message_type, 1bytes
			header[headerOffset] = msg.Header.MessageType
			headerOffset++

			// stream id, 4 bytes, little-endian
			binary.LittleEndian.PutUint32(header[headerOffset:headerOffset+4], msg.Header.StreamID)
			headerOffset += 4

			// chunk extended timestamp header, 0 or 4 bytes, big-endian
			if timestamp >= pt.RtmpExtendTimeStamp {
				binary.BigEndian.PutUint32(header[headerOffset:headerOffset+4], uint32(timestamp))
				headerOffset += 4
			}

		} else {
			// write no message header chunk stream, fmt is 3
			// @remark, if perfer_cid > 0x3F, that is, use 2B/3B chunk header,
			// rollback to 1B chunk header.

			// fmt is 3
			header[headerOffset] = 0xc0 | uint8(msg.Header.PerferCsid&0x3f)
			headerOffset++

			// chunk extended timestamp header, 0 or 4 bytes, big-endian
			// 6.1.3. Extended Timestamp
			// This field is transmitted only when the normal time stamp in the
			// chunk message header is set to 0x00ffffff. If normal time stamp is
			// set to any value less than 0x00ffffff, this field MUST NOT be
			// present. This field MUST NOT be present if the timestamp field is not
			// present. Type 3 chunks MUST NOT have this field.
			// adobe changed for Type3 chunk:
			//        FMLE always sendout the extended-timestamp,
			//        must send the extended-timestamp to FMS,
			//        must send the extended-timestamp to flash-player.
			timestamp := msg.Header.Timestamp
			if timestamp >= pt.RtmpExtendTimeStamp {
				binary.BigEndian.PutUint32(header[headerOffset:headerOffset+4], uint32(timestamp))
				headerOffset += 4
			}
		}

		// not use writev method, we use net.Buffers to mock writev in c/c++, socket disconnected qickly
		if true {
			// send header
			if err = rc.tcpConn.SendBytes(header[:headerOffset]); err != nil {
				log.Println("send msg header failed.")
				return
			}

			//payload
			payloadSize := msg.Header.PayloadLength - payloadOffset
			if payloadSize > rc.outChunkSize {
				payloadSize = rc.outChunkSize
			}

			if err = rc.tcpConn.SendBytes(msg.Payload.Payload[payloadOffset : payloadOffset+payloadSize]); err != nil {
				log.Println("send msg payload failed.")
				return
			}

			payloadOffset += payloadSize
		}
	}

	return
}
