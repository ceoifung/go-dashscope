package audio

import (
	"fmt"
	"io"
	"runtime"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

type Recorder struct {
	reader *io.PipeReader
	writer *io.PipeWriter

	mmde *wca.IMMDeviceEnumerator
	dev  *wca.IMMDevice
	ac   *wca.IAudioClient
	acc  *wca.IAudioCaptureClient

	closeCh chan struct{}
	doneCh  chan struct{}
}

func NewRecorder() (*Recorder, error) {
	r := &Recorder{
		closeCh: make(chan struct{}),
		doneCh:  make(chan struct{}),
	}
	r.reader, r.writer = io.Pipe()

	// IMPORTANT: Use MTA to avoid thread-affinity issues with Go goroutines
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		oleCode := err.(*ole.OleError).Code()
		if oleCode != 0x80010106 && oleCode != 0x00000001 {
			fmt.Printf("Audio: CoInitializeEx warning: %v\n", err)
		}
	}

	if err := r.init(); err != nil {
		r.Close()
		return nil, err
	}

	return r, nil
}

func (r *Recorder) init() error {
	var err error
	if err = wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &r.mmde); err != nil {
		return fmt.Errorf("CoCreateInstance failed: %w", err)
	}

	if err = r.mmde.GetDefaultAudioEndpoint(wca.ECapture, wca.EConsole, &r.dev); err != nil {
		return fmt.Errorf("GetDefaultAudioEndpoint failed: %w", err)
	}

	if err = r.dev.Activate(wca.IID_IAudioClient, wca.CLSCTX_ALL, nil, &r.ac); err != nil {
		return fmt.Errorf("Activate failed: %w", err)
	}

	var wfx *wca.WAVEFORMATEX
	if err = r.ac.GetMixFormat(&wfx); err != nil {
		return fmt.Errorf("GetMixFormat failed: %w", err)
	}

	// Configure for 16kHz, 1 channel, 16-bit PCM
	wfx.WFormatTag = wca.WAVE_FORMAT_PCM
	wfx.NChannels = 1
	wfx.NSamplesPerSec = 16000
	wfx.WBitsPerSample = 16
	wfx.NBlockAlign = 2         // 1 * 16 / 8
	wfx.NAvgBytesPerSec = 32000 // 16000 * 2
	wfx.CbSize = 0

	// Initialize with AUTOCONVERTPCM
	// Buffer duration: 200ms (200 * 10000 units of 100ns)
	bufferDuration := wca.REFERENCE_TIME(200 * 10000)
	err = r.ac.Initialize(
		wca.AUDCLNT_SHAREMODE_SHARED,
		wca.AUDCLNT_STREAMFLAGS_AUTOCONVERTPCM,
		bufferDuration,
		0,
		wfx,
		nil,
	)
	if err != nil {
		return fmt.Errorf("Initialize failed: %w", err)
	}

	if err = r.ac.GetService(wca.IID_IAudioCaptureClient, &r.acc); err != nil {
		return fmt.Errorf("GetService failed: %w", err)
	}

	return nil
}

func (r *Recorder) Start() error {
	if err := r.ac.Start(); err != nil {
		return fmt.Errorf("Start failed: %w", err)
	}

	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(r.doneCh)
		r.captureLoop()
	}()
	return nil
}

func (r *Recorder) captureLoop() {
	var err error
	var pData *byte
	var numFramesToRead uint32
	var flags uint32

	// We configured 16-bit mono, so 2 bytes per frame
	const bytesPerFrame = 2

	ticker := time.NewTicker(20 * time.Millisecond) // Poll every 20ms
	defer ticker.Stop()

	for {
		select {
		case <-r.closeCh:
			return
		case <-ticker.C:
			// Read loop
			for {
				select {
				case <-r.closeCh:
					return
				default:
				}

				var packetLength uint32
				if err = r.acc.GetNextPacketSize(&packetLength); err != nil {
					// fmt.Printf("GetNextPacketSize error: %v\n", err)
					break
				}

				if packetLength == 0 {
					break
				}

				if err = r.acc.GetBuffer(&pData, &numFramesToRead, &flags, nil, nil); err != nil {
					fmt.Printf("Audio: GetBuffer error: %v\n", err)
					break
				}

				if numFramesToRead == 0 {
					r.acc.ReleaseBuffer(numFramesToRead)
					break
				}

				// Copy data
				size := int(numFramesToRead) * bytesPerFrame

				// Create a slice backed by the C memory
				// unsafe.Slice is available in Go 1.17+.
				// Since we are not sure about version, use the verbose way but careful
				var data []byte
				if size > 0 {
					data = (*[1 << 30]byte)(unsafe.Pointer(pData))[:size:size]
				}

				// Write to pipe
				if _, err := r.writer.Write(data); err != nil {
					// Pipe closed or error
					r.acc.ReleaseBuffer(numFramesToRead)
					return
				}

				if err = r.acc.ReleaseBuffer(numFramesToRead); err != nil {
					fmt.Printf("Audio: ReleaseBuffer error: %v\n", err)
					break
				}
			}
		}
	}
}

func (r *Recorder) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

func (r *Recorder) Close() error {
	select {
	case <-r.closeCh:
		// Already closed
		return nil
	default:
		close(r.closeCh)
	}

	// Wait for capture loop to exit
	<-r.doneCh

	if r.ac != nil {
		r.ac.Stop()
	}
	r.writer.Close() // Close pipe writer to unblock reader

	// Release COM objects
	if r.acc != nil {
		r.acc.Release()
		r.acc = nil
	}
	if r.ac != nil {
		r.ac.Release()
		r.ac = nil
	}
	if r.dev != nil {
		r.dev.Release()
		r.dev = nil
	}
	if r.mmde != nil {
		r.mmde.Release()
		r.mmde = nil
	}

	ole.CoUninitialize()
	return nil
}
