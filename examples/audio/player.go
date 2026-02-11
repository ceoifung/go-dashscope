package audio

import (
	"fmt"
	"runtime"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

type Player struct {
	mmde *wca.IMMDeviceEnumerator
	dev  *wca.IMMDevice
	ac   *wca.IAudioClient
	arc  *wca.IAudioRenderClient

	wfx *wca.WAVEFORMATEX
}

func NewPlayer() (*Player, error) {
	// Use MULTITHREADED (MTA) for Go environment
	if err := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); err != nil {
		oleCode := err.(*ole.OleError).Code()
		if oleCode != 0x80010106 && oleCode != 0x00000001 {
			return nil, err
		}
	}

	p := &Player{}
	if err := p.init(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Player) init() error {
	var err error
	if err = wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &p.mmde); err != nil {
		return fmt.Errorf("CoCreateInstance failed: %w", err)
	}

	if err = p.mmde.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &p.dev); err != nil {
		return fmt.Errorf("GetDefaultAudioEndpoint failed: %w", err)
	}

	if err = p.dev.Activate(wca.IID_IAudioClient, wca.CLSCTX_ALL, nil, &p.ac); err != nil {
		return fmt.Errorf("Activate failed: %w", err)
	}

	if err = p.ac.GetMixFormat(&p.wfx); err != nil {
		return fmt.Errorf("GetMixFormat failed: %w", err)
	}

	// Force 16kHz, 1 channel, 16-bit PCM (standard for DashScope TTS)
	p.wfx.WFormatTag = wca.WAVE_FORMAT_PCM
	p.wfx.NChannels = 1
	p.wfx.NSamplesPerSec = 16000
	p.wfx.WBitsPerSample = 16
	p.wfx.NBlockAlign = 2
	p.wfx.NAvgBytesPerSec = 32000
	p.wfx.CbSize = 0

	fmt.Printf("Audio: Attempting to initialize with 16kHz 16-bit Mono PCM (AUTOCONVERTPCM)\n")

	// Buffer duration: 200ms
	bufferDuration := wca.REFERENCE_TIME(200 * 10000)
	err = p.ac.Initialize(
		wca.AUDCLNT_SHAREMODE_SHARED,
		wca.AUDCLNT_STREAMFLAGS_AUTOCONVERTPCM,
		bufferDuration,
		0,
		p.wfx,
		nil,
	)
	if err != nil {
		fmt.Printf("Audio: Initializing with 16kHz failed, falling back to MixFormat: %v\n", err)
		// Fallback: Get MixFormat again and use it
		if err = p.ac.GetMixFormat(&p.wfx); err != nil {
			return fmt.Errorf("GetMixFormat failed: %w", err)
		}
		fmt.Printf("Audio: MixFormat is %d Hz, %d channels\n", p.wfx.NSamplesPerSec, p.wfx.NChannels)
		err = p.ac.Initialize(
			wca.AUDCLNT_SHAREMODE_SHARED,
			0,
			bufferDuration,
			0,
			p.wfx,
			nil,
		)
		if err != nil {
			return fmt.Errorf("Initialize with MixFormat failed: %w", err)
		}
	} else {
		fmt.Printf("Audio: Successfully initialized with 16kHz Mono\n")
	}
	if err != nil {
		return fmt.Errorf("Initialize failed: %w", err)
	}

	if err = p.ac.GetService(wca.IID_IAudioRenderClient, &p.arc); err != nil {
		return fmt.Errorf("GetService failed: %w", err)
	}

	return nil
}

func (p *Player) Start() error {
	return p.ac.Start()
}

func (p *Player) Stop() error {
	return p.ac.Stop()
}

func (p *Player) Close() {
	if p.ac != nil {
		p.ac.Stop()
	}
	if p.mmde != nil {
		p.mmde.Release()
	}
	if p.dev != nil {
		p.dev.Release()
	}
	if p.ac != nil {
		p.ac.Release()
	}
	if p.arc != nil {
		p.arc.Release()
	}
	ole.CoUninitialize()
}

// Play plays the given PCM data. It blocks until the data is written to the buffer.
func (p *Player) Play(data []byte) error {
	// Ensure Play runs on a consistent thread for Windows COM
	// Though MTA is used, some driver-level calls prefer consistency
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var err error
	var bufferSize uint32
	var padding uint32
	var available uint32
	var buffer *byte

	if err = p.ac.GetBufferSize(&bufferSize); err != nil {
		return err
	}

	offset := 0
	total := len(data)

	for offset < total {
		if err = p.ac.GetCurrentPadding(&padding); err != nil {
			return err
		}
		available = bufferSize - padding
		if available == 0 {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		toWrite := uint32(total - offset)
		framesToWrite := toWrite / uint32(p.wfx.NBlockAlign)
		if framesToWrite > available {
			framesToWrite = available
		}
		if framesToWrite == 0 {
			continue
		}

		if err = p.arc.GetBuffer(framesToWrite, &buffer); err != nil {
			return err
		}

		bytesToWrite := framesToWrite * uint32(p.wfx.NBlockAlign)
		// Copy data to buffer
		// Unsafe copy because wca returns *byte
		// We need to copy data[offset : offset+bytesToWrite] to buffer
		src := data[offset : offset+int(bytesToWrite)]
		for i := 0; i < len(src); i++ {
			*(*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(buffer)) + uintptr(i))) = src[i]
		}

		if err = p.arc.ReleaseBuffer(framesToWrite, 0); err != nil {
			return err
		}

		offset += int(bytesToWrite)
	}
	return nil
}
