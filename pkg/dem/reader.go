package dem

import (
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/hungys/go-lz4"
	"github.com/wladich/elevation_server/pkg/constants"
	"io"
	"os"
)

type StorageReader storageAbstract

func NewReader(path string) (*StorageReader, error) {
	var storage StorageReader
	idxPath := path + ".idx"
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	storage.fData = f

	storage.index = &tileFileIndex{}
	f, err = os.Open(idxPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	decoder := gob.NewDecoder(f)
	if err = decoder.Decode(storage.index); err!= nil {
		return nil, err
	}

	return &storage, nil
}

func decompressTile(compressed []byte) (*TileRawData, error) {
	var tileData TileRawData
	n, err := lz4.DecompressSafe(compressed, tileData[:])
	if n != constants.TileBytes {
		return nil, errors.New(fmt.Sprintf("Unexpected tile size: %v", n))
	}
	if err != nil {
		return nil, err
	}
	return &tileData, nil
}

func (storage *StorageReader) GetTile(index TileIndex) (*Tile, error) {
	x := index.X + 180*constants.HgtSplitParts
	y := index.Y + 90*constants.HgtSplitParts
	if x < 0 || y < 0 || x > len(storage.index) || y > len(storage.index[x]) {
		return nil, nil
	}
	tileFileIndex := storage.index[x][y]
	if tileFileIndex.Size == 0 {
		return nil, nil
	}
	if _, err := storage.fData.Seek(tileFileIndex.Offset, io.SeekStart); err != nil {
		return nil, err
	}
	compressed := make([]byte, tileFileIndex.Size)
	n, err := storage.fData.Read(compressed)
	if err != nil {
		return nil, err
	}
	if int64(n) != tileFileIndex.Size {
		return nil, errors.New("tile data incomplete")
	}
	tileData, err := decompressTile(compressed)
	if err != nil {
		return nil, err
	}
	tile := tileFromRaw(TileRaw{*tileData, index})
	return &tile, nil
}

func (storage *StorageReader) Close() error {
	return storage.fData.Close()
}
