package repository

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
)

type Index struct {
	header  *Header
	entries []*Entry
}

// Header represents the 12 byte header of the Git index file.
type Header struct {
	Signature  [4]byte
	Version    uint32
	NumEntries uint32
}

// Entry represents a single entry in the Git index file.
type Entry struct {
	// creation time
	cTimeSec  uint32
	cTimeNano uint32

	// modification time
	mTimeSec  uint32
	mTimeNano uint32

	// device number where the file is located
	dev uint32
	// inode number -> used to store metadata, disk loc etc
	inode uint32
	mode  uint32
	uid   uint32
	gid   uint32
	size  uint32
	sha   [20]byte
	flags uint16
	path  string
}

type IndexFlags struct {
	assumeValid bool // skip validation checks if set
	extended    bool // entry might have additional metadata if set
	stage       uint16
}

func parseFlags(flags uint16) IndexFlags {
	return IndexFlags{
		// bit 15
		assumeValid: (flags & 0b1000000000000000) != 0,
		// bit 14
		extended: (flags & 0b0100000000000000) != 0,
		// bit 13, 12 and right shift by 12
		stage: (flags & 0b0011000000000000) >> 12,
	}
}

func calcPadding(n int) int {
	// first, reserve 2 bytes for variable path length
	// this can be gotten from namelen, encoded in the flags
	// representing length of the path
	// apparently namelen provides some kind of optimization
	// so that length of the path could be accessed directly
	// without extra calculations
	// these 2 bytes have ruined almost 2 days of mine
	baseLen := (n - 2) / 8

	// move to next 8 byte boundary
	alignedBoundary := (baseLen+1)*8 + 2
	return alignedBoundary - n
}

func parseIndex(path string) (*Index, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Couldn't read index file: %w", err)
	}

	var index Index
	header := &Header{}
	reader := bytes.NewReader(contents[:12])
	err = binary.Read(reader, binary.BigEndian, header)
	if err != nil {
		return nil, fmt.Errorf("Couldn't parse index header: %w", err)
	}

	index.header = header
	numEntries := header.NumEntries

	offset := 12
	for i := 0; i < int(numEntries); i++ {
		entry, bytesRead, err := parseEntry(contents[offset:])
		if err != nil {
			return nil, fmt.Errorf("Error parsing entry %d: %w", i+1, err)
		}

		offset += bytesRead
		index.entries = append(index.entries, entry)
	}

	return &index, nil
}

func parseEntry(data []byte) (*Entry, int, error) {
	var fixedEntry struct {
		CTimeSeconds uint32
		CTimeNanosec uint32
		MTimeSeconds uint32
		MTimeNanosec uint32
		Dev          uint32
		Inode        uint32
		Mode         uint32
		Uid          uint32
		Gid          uint32
		Size         uint32
		Sha          [20]byte
		Flags        uint16
	}

	reader := bytes.NewReader(data)
	err := binary.Read(reader, binary.BigEndian, &fixedEntry)
	if err != nil {
		return nil, 0, err
	}

	// bits 0-11 represent the length of path
	pathLen := int(fixedEntry.Flags & 0x0FFF)
	if pathLen <= 0 || pathLen > len(data)-62 {
		return nil, 0, fmt.Errorf("Invalid path length %d", pathLen)
	}

	path := make([]byte, pathLen)
	_, err = reader.Read(path)
	if err != nil {
		return nil, 0, err
	}

	entrySize := 62 + pathLen
	padding := calcPadding(pathLen)
	totalBytesRead := entrySize + padding

	entry := &Entry{
		cTimeSec:  fixedEntry.CTimeSeconds,
		cTimeNano: fixedEntry.CTimeNanosec,
		mTimeSec:  fixedEntry.MTimeSeconds,
		mTimeNano: fixedEntry.MTimeNanosec,
		dev:       fixedEntry.Dev,
		inode:     fixedEntry.Inode,
		mode:      fixedEntry.Mode,
		uid:       fixedEntry.Uid,
		gid:       fixedEntry.Gid,
		size:      fixedEntry.Size,
		sha:       fixedEntry.Sha,
		path:      string(path),
		flags:     fixedEntry.Flags,
	}

	return entry, totalBytesRead, nil
}
