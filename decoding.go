package tiled

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/klauspost/compress/zstd"
)

const (
	FlipHorizontalFlag uint32 = 0x80000000
	FlipVerticalFlag   uint32 = 0x40000000
	FlipDiagonalFlag   uint32 = 0x20000000
	RotateHexFlag      uint32 = 0x10000000
	GIDMask            uint32 = 0x1FFFFFFF
)

func DecodeGID(gid uint32) (tileID uint32, flags FlipFlag) {
	tileID = gid & GIDMask
	if gid&FlipHorizontalFlag != 0 {
		flags |= FlipHorizontal
	}
	if gid&FlipVerticalFlag != 0 {
		flags |= FlipVertical
	}
	if gid&FlipDiagonalFlag != 0 {
		flags |= FlipDiagonal
		if flags&(FlipHorizontal|FlipVertical) != 0 {
			flags ^= FlipHorizontal | FlipVertical
		}
	}
	return
}

func DecodeContent(content string, encoding Encoding, compression Compression) ([]uint32, error) {
	switch encoding {
	case EncodingCSV:
		return decodeCSV(content)

	case EncodingBase64:
		return decodeBase64(content, compression)
	}
	// Note: XML encoding is not supported.
	panic(fmt.Sprintf("unsupported encoding: %s", encoding))
}

func decodeCSV(content string) ([]uint32, error) {
	var data []uint32
	for s := range strings.SplitSeq(content, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		tileIndex, err := strconv.Atoi(s)
		if err != nil {
			return nil, fmt.Errorf("invalid CSV layer data: %w", err)
		}
		data = append(data, uint32(tileIndex))
	}
	return data, nil
}

func decodeBase64(content string, compression Compression) ([]uint32, error) {
	var decoded []byte
	var err error

	decoded, err = decodeBase64Content(content)
	if err != nil {
		return nil, err
	}

	if compression != CompressionNone {
		switch compression {
		case CompressionGzip:
			decoded, err = decompressGzip(decoded)
			if err != nil {
				return nil, err
			}
		case CompressionZlib:
			decoded, err = decompressZlib(decoded)
			if err != nil {
				return nil, err
			}
		case CompressionZstd:
			decoded, err = decompressZstd(decoded)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unsupported compression: %s", compression)
		}
	}

	if len(decoded)%4 != 0 {
		return nil, fmt.Errorf("invalid base64 layer data length: %d", len(decoded))
	}

	data := make([]uint32, len(decoded)/4)
	for i := range data {
		data[i] = uint32(decoded[i*4]) | uint32(decoded[i*4+1])<<8 | uint32(decoded[i*4+2])<<16 | uint32(decoded[i*4+3])<<24
	}

	return data, nil
}

func decodeBase64Content(content string) ([]byte, error) {
	trimmed := strings.TrimSpace(content)

	decoded, err := base64.StdEncoding.DecodeString(trimmed)
	if err != nil {
		return nil, err
	}

	return decoded, nil
}

func decompress(data []byte, decompressFunc func(io.Reader) (io.ReadCloser, error)) ([]byte, error) {
	reader, err := decompressFunc(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	var decompressed bytes.Buffer
	_, err = io.Copy(&decompressed, reader)
	if err != nil {
		return nil, err
	}

	return decompressed.Bytes(), nil
}

func decompressGzip(data []byte) ([]byte, error) {
	return decompress(data, func(r io.Reader) (io.ReadCloser, error) {
		return gzip.NewReader(r)
	})
}

func decompressZlib(data []byte) ([]byte, error) {
	return decompress(data, func(r io.Reader) (io.ReadCloser, error) {
		return zlib.NewReader(r)
	})
}

func decompressZstd(data []byte) ([]byte, error) {
	decoder, err := zstd.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer decoder.Close()

	var decompressed bytes.Buffer
	_, err = io.Copy(&decompressed, decoder)
	if err != nil {
		return nil, err
	}

	return decompressed.Bytes(), nil
}
