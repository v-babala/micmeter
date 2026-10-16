<div align="center">

# 🎙️ MicMeter

**A clean, real-time microphone level meter for your terminal.**

Measure your microphone input in **dBFS** with a live visual meter, peak readout, and keyboard-driven device selection — powered by FFmpeg.

[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![Platforms](https://img.shields.io/badge/platforms-Windows%20%7C%20macOS%20%7C%20Linux-555)](#%EF%B8%8F-platform-support)
[![ffmpeg](https://img.shields.io/badge/requires-ffmpeg-007808?logo=ffmpeg&logoColor=white)](https://ffmpeg.org)
[![License](https://img.shields.io/badge/license-MIT-blue)](#-license)

</div>

---

## ✨ Features

- **Live dBFS meter** — monitor the current RMS level while speaking.
- **Peak hold** — keep an eye on the highest level seen during the current capture session.
- **Device picker** — browse available Windows audio inputs without typing device names.
- **Terminal-native UI** — responsive layout, colored meter, and keyboard controls, built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss).
- **FFmpeg-powered** — uses DirectShow for device discovery and capture; no Go audio driver dependency.
- **Small and portable** — build a standalone executable; FFmpeg must be installed separately and available on `PATH`.

## 📸 Interface

The app includes an audio input list, a live meter, current RMS level, peak level, and a status line. The meter spans approximately **−60 to 0 dBFS**; values near 0 dBFS indicate that the input is close to digital clipping.

## 🚀 Requirements

- Windows 10 or Windows 11
- [Go 1.22+](https://go.dev/dl/) to build from source
- [FFmpeg](https://ffmpeg.org/download.html) with the **DirectShow (`dshow`) input device** enabled
- A microphone or other Windows audio input device

Check that FFmpeg is available:

```powershell
ffmpeg -version
```

## 📦 Installation

### Build from source

```powershell
git clone https://github.com/yourname/micmeter.git
cd micmeter
go mod tidy
go build -o micmeter.exe .
.\micmeter.exe
```

Replace `yourname` in the repository URL and in `go.mod` with your GitHub username before publishing.

### Build a Windows executable

From a Windows machine, run:

```powershell
go build -trimpath -ldflags="-s -w" -o micmeter.exe .
```

You can then run `micmeter.exe` from any terminal, as long as `ffmpeg.exe` is on `PATH`.

## 🎛️ Usage

1. Launch `micmeter.exe`.
2. Use **↑ / ↓** or **J / K** to choose an audio input.
3. Press **Enter** to start monitoring.
4. Speak into the microphone and watch the live RMS meter and peak readout.
5. Press **Enter** or **Esc** to stop monitoring; press **R** to refresh the device list.
6. Press **Q** or **Ctrl+C** to exit.

## ⌨️ Keyboard shortcuts

| Key | Action |
|---|---|
| `↑` / `↓`, `J` / `K` | Select an audio input |
| `Enter` / `Space` | Start or stop monitoring |
| `R` | Refresh audio devices |
| `Esc` / `Backspace` | Stop monitoring |
| `Q` / `Ctrl+C` | Quit |

## 📊 Understanding dBFS

**dBFS** means *decibels relative to full scale*, the digital audio system's maximum representable level.

- **−60 to −30 dBFS:** quiet input or background noise
- **−30 to −18 dBFS:** moderate input
- **−18 to −6 dBFS:** strong input
- **−6 to 0 dBFS:** close to the digital ceiling; watch for clipping

These are rough monitoring ranges, not strict recording targets. The displayed value is the **RMS level** reported by FFmpeg's `astats` filter, not a calibrated sound-pressure measurement in dB SPL. Microphone gain, distance, processing, and the audio device all affect the reading.

## 🛠️ Troubleshooting

### `ffmpeg` is not recognized

Install FFmpeg and add the folder containing `ffmpeg.exe` to your Windows `PATH`. Open a new terminal and verify with `ffmpeg -version`.

### No audio devices found

Confirm Windows can see the microphone in **Settings → System → Sound → Input**. Then check that your FFmpeg build supports DirectShow:

```powershell
ffmpeg -devices
```

### The meter does not move

Make sure the correct input is selected, that the microphone is not muted, and that Windows input permissions allow desktop apps to access the microphone. Check the input level in Windows Sound settings as well.

### Another application is using the microphone

Some drivers allow shared capture; others may be configured for exclusive access. Close other recording applications and try again.

## 🧱 Tech stack

- [Go](https://go.dev/)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — terminal UI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — terminal styling
- [FFmpeg](https://ffmpeg.org/) — device discovery, audio capture, and RMS analysis

## 🗺️ Roadmap

- [ ] Configurable meter range and peak reset
- [ ] Optional dBFS threshold indicators
- [ ] Cross-platform backends (ALSA / PulseAudio / PipeWire / AVFoundation)
- [ ] Configurable FFmpeg executable path
- [ ] Release binaries via GitHub Actions

## 📄 License

MIT — see [LICENSE](LICENSE).
