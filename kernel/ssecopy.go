package kernel

import (
	"bytes"
	"context"
	"io"
	"net/http"
)

// copyOutput is the Template Method: one SSE frame loop, Strategy chooses emit.
func copyOutput(ctx context.Context, dst http.ResponseWriter, src io.Reader, cfg Config, inspectors []Inspector) (InspectResult, error) {
	if cfg.policyMode() == PolicyEnforce && NeedsOutputHoldback(inspectors) {
		return copyHoldbackRedact(ctx, dst, src, cfg, inspectors)
	}
	return copyFlushScan(ctx, dst, src, cfg, inspectors)
}

// copySSEFrames splits src on '\n' and yields each frame (including the newline).
func copySSEFrames(src io.Reader, onFrame func(frame []byte) error, onEOF func(tail []byte) error) error {
	var pending []byte
	buf := make([]byte, 4096)
	for {
		n, err := src.Read(buf)
		if n > 0 {
			pending = append(pending, buf[:n]...)
			for {
				i := bytes.IndexByte(pending, '\n')
				if i < 0 {
					break
				}
				frame := pending[:i+1]
				pending = pending[i+1:]
				if herr := onFrame(frame); herr != nil {
					return herr
				}
			}
		}
		if err == io.EOF {
			return onEOF(pending)
		}
		if err != nil {
			return err
		}
	}
}

func copyFlushScan(ctx context.Context, dst http.ResponseWriter, src io.Reader, cfg Config, inspectors []Inspector) (InspectResult, error) {
	var last InspectResult
	flusher, _ := dst.(http.Flusher)
	win := newSSEWindow(defaultWindowBytes)

	onFrame := func(frame []byte) error {
		if err := writeAndFlush(dst, flusher, frame); err != nil {
			return err
		}
		if len(inspectors) == 0 {
			return nil
		}
		for _, snap := range win.Feed(frame) {
			hit := InspectOutputWindow(ctx, inspectors, snap)
			if hit.EngineError != "" {
				last = hit
				if cfg.failClosed() {
					return errOutputBlocked
				}
				continue
			}
			if hit.Blocks() {
				last = hit
				if cfg.policyMode() == PolicyEnforce {
					return errOutputBlocked
				}
			} else if hit.Hit {
				last = hit
			}
		}
		return nil
	}

	err := copySSEFrames(src, onFrame, func(tail []byte) error {
		if len(tail) == 0 {
			return nil
		}
		return writeAndFlush(dst, flusher, tail)
	})
	if err == errOutputBlocked {
		return last, nil
	}
	return last, err
}
