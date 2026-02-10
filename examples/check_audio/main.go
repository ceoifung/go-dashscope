package main

import (
	"fmt"
	"os"

	"github.com/go-ole/go-ole"
	"github.com/moutend/go-wca/pkg/wca"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		return err
	}
	defer ole.CoUninitialize()

	var mmde *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &mmde); err != nil {
		return err
	}
	defer mmde.Release()

	var dev *wca.IMMDevice
	if err := mmde.GetDefaultAudioEndpoint(wca.ECapture, wca.EConsole, &dev); err != nil {
		return err
	}
	defer dev.Release()

	var ac *wca.IAudioClient
	if err := dev.Activate(wca.IID_IAudioClient, wca.CLSCTX_ALL, nil, &ac); err != nil {
		return err
	}
	defer ac.Release()

	var wfx *wca.WAVEFORMATEX
	if err := ac.GetMixFormat(&wfx); err != nil {
		return err
	}

	fmt.Printf("Default Mix Format:\n")
	fmt.Printf("  Format Tag: %d\n", wfx.WFormatTag)
	fmt.Printf("  Channels: %d\n", wfx.NChannels)
	fmt.Printf("  Samples Per Sec: %d\n", wfx.NSamplesPerSec)
	fmt.Printf("  Bits Per Sample: %d\n", wfx.WBitsPerSample)

	// Try to initialize with desired format
	desiredWfx := *wfx // Copy
	desiredWfx.WFormatTag = wca.WAVE_FORMAT_PCM
	desiredWfx.NChannels = 1
	desiredWfx.NSamplesPerSec = 16000
	desiredWfx.WBitsPerSample = 16
	desiredWfx.NBlockAlign = 2
	desiredWfx.NAvgBytesPerSec = 32000
	desiredWfx.CbSize = 0

	fmt.Println("Attempting to initialize with 16kHz Mono 16bit PCM...")
	// AUDCLNT_STREAMFLAGS_AUTOCONVERTPCM | AUDCLNT_STREAMFLAGS_EVENTCALLBACK (if needed)
	// For loopback/capture, AUTOCONVERTPCM applies to capture client too on newer Windows.

	// Note: AUTOCONVERTPCM = 0x80000000.
	// We need to use shared mode.
	err := ac.Initialize(
		wca.AUDCLNT_SHAREMODE_SHARED,
		wca.AUDCLNT_STREAMFLAGS_AUTOCONVERTPCM, // Try with auto convert
		10000000,                               // 1 second buffer duration (100ns units)
		0,
		&desiredWfx,
		nil,
	)

	if err != nil {
		fmt.Printf("Initialize failed: %v\n", err)
		// Try without AUTOCONVERTPCM just to see (likely fail)
	} else {
		fmt.Println("Initialize SUCCESS!")
	}

	return nil
}
