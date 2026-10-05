package record

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

// RecorderOptions configures snapshot capture intervals, limits, and destination.
type RecorderOptions struct {
	Output    io.Writer
	FilePath  string
	Interval  time.Duration
	Count     int
	Duration  time.Duration
	Collector CollectorFunc
	OnSample  func(snap *model.Snapshot)
}

// RunRecorder streams periodic system snapshots until count, duration, or context cancellation occurs.
func RunRecorder(ctx context.Context, opts RecorderOptions) error {
	if opts.Interval <= 0 {
		opts.Interval = 2 * time.Second
	}
	if opts.Collector == nil {
		opts.Collector = DefaultCollector
	}

	var w io.Writer = opts.Output
	var fileCloser io.Closer
	if w == nil {
		if opts.FilePath == "" {
			return errors.New("must specify output writer or file path")
		}
		f, err := os.OpenFile(opts.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("open output file %q: %w", opts.FilePath, err)
		}
		fileCloser = f
		w = f
	}
	defer func() {
		if fileCloser != nil {
			_ = fileCloser.Close()
		}
	}()

	writer := NewWriter(w)

	// Capture initial sample immediately
	snap, err := opts.Collector(0)
	if err != nil {
		return fmt.Errorf("collect initial sample: %w", err)
	}
	if err := writer.Write(snap); err != nil {
		return fmt.Errorf("write initial sample: %w", err)
	}
	if opts.OnSample != nil {
		opts.OnSample(snap)
	}

	sampleIndex := 1
	if opts.Count > 0 && sampleIndex >= opts.Count {
		return writer.Flush()
	}

	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()

	var timeoutChan <-chan time.Time
	if opts.Duration > 0 {
		timer := time.NewTimer(opts.Duration)
		defer timer.Stop()
		timeoutChan = timer.C
	}

	for {
		select {
		case <-ctx.Done():
			_ = writer.Flush()
			return ctx.Err()

		case <-timeoutChan:
			_ = writer.Flush()
			return nil

		case <-ticker.C:
			snap, err := opts.Collector(sampleIndex)
			if err != nil {
				return fmt.Errorf("collect sample %d: %w", sampleIndex, err)
			}
			if err := writer.Write(snap); err != nil {
				return fmt.Errorf("write sample %d: %w", sampleIndex, err)
			}
			if opts.OnSample != nil {
				opts.OnSample(snap)
			}
			sampleIndex++
			if opts.Count > 0 && sampleIndex >= opts.Count {
				return writer.Flush()
			}
		}
	}
}
