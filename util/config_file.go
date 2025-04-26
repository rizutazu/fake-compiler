package util

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"io"
	"os"
)

const CONFIG_MAGIC = 0xfcfcfcfc

// config file spec
// everything in little-endian
type Config struct {
	magic               uint32 // should be const CONFIG_MAGIC
	TypeLength          uint32 // length of CompilerType, which is ascii string, does NOT include NULL terminator
	CompilerType        string
	UncompressedContent []byte
}

func ParseConfigFile(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) > 0xffffffff {
		return nil, errors.New("config file: file size too large")
	}
	if len(b) < 8 {
		return nil, errors.New("config file: invalid format")
	}
	magic := binary.LittleEndian.Uint32(b)
	if magic != CONFIG_MAGIC {
		return nil, errors.New("config file: magic does not match")
	}
	typeLen := binary.LittleEndian.Uint32(b[4:])
	if uint32(len(b)) < typeLen+8 {
		return nil, errors.New("config file: malformed compiler type")
	}
	typeByte := b[8 : 8+typeLen]
	compressedContent := b[8+typeLen:]
	w, err := gzip.NewReader(bytes.NewReader(compressedContent))
	if err != nil {
		return nil, err
	}
	uncompressedContent, err := io.ReadAll(w)
	if err != nil {
		return nil, err
	}
	return &Config{
		magic:               CONFIG_MAGIC,
		TypeLength:          typeLen,
		CompilerType:        string(typeByte),
		UncompressedContent: uncompressedContent,
	}, nil
}

func DumpConfigFile(path string, compilerType []byte, uncompressedContent []byte) error {
	// open file
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := make([]byte, 4)

	// magic
	binary.LittleEndian.PutUint32(buf, CONFIG_MAGIC)
	_, err = f.Write(buf)
	if err != nil {
		return err
	}

	// TypeLength
	if len(compilerType) > 0xffffffff {
		return errors.New("config file: compiler type is too large")
	}
	binary.LittleEndian.PutUint32(buf, uint32(len(compilerType)))
	_, err = f.Write(buf)
	if err != nil {
		return err
	}
	// compilerType
	_, err = f.Write(compilerType)
	if err != nil {
		return err
	}
	// CompressedContent
	w := gzip.NewWriter(f)
	defer w.Close()
	_, err = w.Write(uncompressedContent)

	return err
}
