// Code originally written by Steve McCoy under the MIT license and altered by
// Jonas747. The bulk of those was removed and the rest rewritten.
//
// © 2020 diamondburned under the ISC license.

// Package oggreader provides a small abstraction to unwrap Opus frames for
// packets. It does not fully implement the Ogg specifications, and it does not
// perform a checksum of each packet.
package oggreader

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// HeaderSize is the header size.
const HeaderSize = 27

// MaxSegmentSize is the maximum segment size.
const MaxSegmentSize = 255

// MaxPacketSize is the maximum packet size.
const MaxPacketSize = MaxSegmentSize * 255

// MaxPageSize is the maximum page size, which is 65307 bytes per the RFC.
const MaxPageSize = HeaderSize + MaxSegmentSize + MaxPacketSize

var oggs = [...]byte{'O', 'g', 'g', 'S'}

// DecodeBuffered decodes using a buffered reader. This could help greatly
// reduce Read calls, which might reduce syscalls if any. The allocated buffer
// will be 65KB plus another 65KB of internal buffer, sized after MaxPageSize.
func DecodeBuffered(dst io.Writer, src io.Reader) error {
	return Decode(dst, bufio.NewReaderSize(src, MaxPageSize))
}

// Decode decodes the given src stream of Opus and writes each packet to dst.
func Decode(dst io.Writer, src io.Reader) error {
	return DecodeBuf(dst, src, make([]byte, MaxPageSize))
}

// DecodeBuf decodes with a custom buffer. The buffer must be at least
// MaxPageSize.
func DecodeBuf(dst io.Writer, src io.Reader, buf []byte) error {
	if len(buf) < MaxPageSize {
		return errors.New("buffer is too small")
	}

	err := decode(dst, src, buf)
	if err == io.EOF {
		return nil
	}
	return err
}

func decode(dst io.Writer, src io.Reader, buffer []byte) error {
	var (
		headerBuf = buffer[:HeaderSize]

		// Packet boundaries.
		ixseg int   = 0
		start int64 = 0
		end   int64 = 0

		header pageHeader
	)

	for {
		if _, err := io.ReadFull(src, headerBuf); err != nil {
			return err
		}

		if !bytes.Equal(headerBuf[:4:4], oggs[:]) {
			return fmt.Errorf("invalid oggs header: %q % x", headerBuf[:4], headerBuf)
		}

		if _, err := header.Read(headerBuf); err != nil {
			return err
		}

		if header.Nsegs < 1 {
			return ErrBadSegs
		}

		nsegs := int(header.Nsegs)
		segTblBuf := buffer[HeaderSize : HeaderSize+nsegs]

		if _, err := io.ReadFull(src, segTblBuf); err != nil {
			return err
		}

		var pageDataLen = 0
		for _, l := range segTblBuf {
			pageDataLen += int(l)
		}

		packetBuf := buffer[HeaderSize+nsegs : HeaderSize+nsegs+pageDataLen]

		if _, err := io.ReadFull(src, packetBuf); err != nil {
			return err
		}

		ixseg = 0
		start = 0
		end = 0

		for {
			for ixseg < nsegs {
				segment := segTblBuf[ixseg]
				end += int64(segment)

				ixseg++

				if segment < 0xFF {
					break
				}
			}

			_, err := dst.Write(packetBuf[start:end])
			if err != nil {
				return fmt.Errorf("failed to write a packet: %w", err)
			}

			if ixseg >= nsegs {
				break
			}

			start = end
		}
	}
}

type pageHeader struct {
	Nsegs byte
}

var byteOrder = binary.LittleEndian

// Read reads b into pageHeader.
func (ph *pageHeader) Read(b []byte) (int, error) {
	if len(b) != HeaderSize {
		return 0, io.ErrUnexpectedEOF
	}

	// We only care about this.
	ph.Nsegs = b[26]

	return HeaderSize, nil
}

// ErrBadSegs is the error used when trying to decode a page with a segment
// table size less than 1.
var ErrBadSegs = errors.New("invalid segment table size")
