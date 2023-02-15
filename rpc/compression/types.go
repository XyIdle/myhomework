package compression

type Compression interface {
	Code() uint8
	Compress(val []byte) ([]byte, error)
	Decompress(data []byte) ([]byte, error)
}
