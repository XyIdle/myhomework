package zstd

import (
	"github.com/klauspost/compress/zstd"
)

type Compressor struct {
}

func (c *Compressor) Code() uint8 {
	return 1
}

func (c *Compressor) Compress(data []byte) ([]byte, error) {
	var encoder, _ = zstd.NewWriter(nil)
	return encoder.EncodeAll(data, make([]byte, 0, len(data))), nil
}

func (c *Compressor) Decompress(data []byte) ([]byte, error) {
	var decoder, _ = zstd.NewReader(nil, zstd.WithDecoderConcurrency(0))
	return decoder.DecodeAll(data, nil)
}
