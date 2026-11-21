package record

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

const (
	// CurrentSnapshotVersion indicates the serialization format version.
	CurrentSnapshotVersion = 1

	// maxScanTokenSize sets maximum line size (64MB) to support large process trees.
	maxScanTokenSize = 64 * 1024 * 1024
)

// Serialize converts a snapshot into a newline-delimited JSON byte slice.
func Serialize(snap *model.Snapshot) ([]byte, error) {
	if snap == nil {
		return nil, errors.New("cannot serialize nil snapshot")
	}
	if snap.Version == 0 {
		snap.Version = CurrentSnapshotVersion
	}
	data, err := json.Marshal(snap)
	if err != nil {
		return nil, fmt.Errorf("serialize snapshot: %w", err)
	}
	return append(data, '\n'), nil
}

// Deserialize decodes a JSON byte slice into a Snapshot.
func Deserialize(data []byte) (*model.Snapshot, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, errors.New("empty snapshot data")
	}
	var snap model.Snapshot
	if err := json.Unmarshal(trimmed, &snap); err != nil {
		return nil, fmt.Errorf("deserialize snapshot: %w", err)
	}
	return &snap, nil
}

// Writer appends serialized snapshots to an underlying io.Writer.
type Writer struct {
	w  io.Writer
	bw *bufio.Writer
}

// NewWriter creates a streaming snapshot writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{
		w:  w,
		bw: bufio.NewWriter(w),
	}
}

// Write writes a single snapshot to the stream followed by a newline.
func (w *Writer) Write(snap *model.Snapshot) error {
	data, err := Serialize(snap)
	if err != nil {
		return err
	}
	if _, err := w.bw.Write(data); err != nil {
		return fmt.Errorf("write snapshot: %w", err)
	}
	return w.bw.Flush()
}

// Flush flushes buffered data to the destination writer.
func (w *Writer) Flush() error {
	return w.bw.Flush()
}

// Reader streams snapshots from an io.Reader.
type Reader struct {
	scanner *bufio.Scanner
	array   []*model.Snapshot
	idx     int
}

// NewReader constructs a snapshot reader supporting JSON lines and JSON arrays.
func NewReader(r io.Reader) (*Reader, error) {
	br := bufio.NewReader(r)

	// Peek first bytes to detect gzip magic number
	magic, err := br.Peek(2)
	var streamReader io.Reader = br
	if err == nil && len(magic) == 2 && magic[0] == 0x1f && magic[1] == 0x8b {
		gz, gzErr := gzip.NewReader(br)
		if gzErr != nil {
			return nil, fmt.Errorf("open gzip reader: %w", gzErr)
		}
		streamReader = gz
	}

	// Read first non-whitespace character to distinguish array from NDJSON
	var headBuf bytes.Buffer
	var firstByte byte
	foundFirst := false
	for {
		b, rErr := streamReader.Read(magic[:1])
		if rErr != nil || b == 0 {
			break
		}
		headBuf.WriteByte(magic[0])
		if magic[0] != ' ' && magic[0] != '\t' && magic[0] != '\r' && magic[0] != '\n' {
			firstByte = magic[0]
			foundFirst = true
			break
		}
	}

	combined := io.MultiReader(&headBuf, streamReader)

	// If the file starts with '[', read as JSON array
	if foundFirst && firstByte == '[' {
		all, readErr := io.ReadAll(combined)
		if readErr != nil {
			return nil, fmt.Errorf("read json array: %w", readErr)
		}
		var snaps []*model.Snapshot
		if err := json.Unmarshal(all, &snaps); err != nil {
			return nil, fmt.Errorf("decode json array: %w", err)
		}
		return &Reader{array: snaps}, nil
	}

	scanner := bufio.NewScanner(combined)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, maxScanTokenSize)

	return &Reader{scanner: scanner}, nil
}

// Read reads the next snapshot from the stream. Returns io.EOF at end of stream.
func (r *Reader) Read() (*model.Snapshot, error) {
	if r.array != nil {
		if r.idx >= len(r.array) {
			return nil, io.EOF
		}
		snap := r.array[r.idx]
		r.idx++
		return snap, nil
	}

	for r.scanner.Scan() {
		line := bytes.TrimSpace(r.scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		snap, err := Deserialize(line)
		if err != nil {
			return nil, err
		}
		return snap, nil
	}

	if err := r.scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan error: %w", err)
	}
	return nil, io.EOF
}

// ReadAll reads all remaining snapshots in the stream.
func (r *Reader) ReadAll() ([]*model.Snapshot, error) {
	var snaps []*model.Snapshot
	for {
		snap, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		snaps = append(snaps, snap)
	}
	return snaps, nil
}

// LoadFile reads all snapshots from the given file path or stdin ("-").
func LoadFile(filePath string) ([]*model.Snapshot, error) {
	if strings.TrimSpace(filePath) == "" {
		return nil, errors.New("missing file path")
	}

	var r io.Reader
	if filePath == "-" {
		r = os.Stdin
	} else {
		f, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("open file %q: %w", filePath, err)
		}
		defer f.Close()
		r = f
	}

	reader, err := NewReader(r)
	if err != nil {
		return nil, err
	}

	snaps, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(snaps) == 0 {
		return nil, fmt.Errorf("no snapshots found in %q", filePath)
	}
	return snaps, nil
}

// SaveFile writes the given snapshots to a file in NDJSON format.
func SaveFile(filePath string, snapshots []*model.Snapshot) error {
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create file %q: %w", filePath, err)
	}
	defer f.Close()

	w := NewWriter(f)
	for _, s := range snapshots {
		if err := w.Write(s); err != nil {
			return err
		}
	}
	return w.Flush()
}
