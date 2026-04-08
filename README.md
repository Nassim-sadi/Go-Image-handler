# ImageHandler

A fast, personal image downloader and processor built with Go and Fyne.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go)
![Fyne](https://img.shields.io/badge/GUI-Fyne%202.7-blue?style=flat-square)
![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)

## Features

- **Download images** from URLs
- **Batch processing** with concurrent downloads
- **Resize images** with multiple modes (fit, exact, cover)
- **Convert formats** (WebP, JPEG, PNG)
- **Live color adjustments** (saturation, brightness, contrast)
- **Preview** before processing
- **Project organization** - organize images by project folders
- **Preset system** - save and reuse processing settings

## Installation

### Pre-built Binary

Download the latest release from [GitHub Releases](https://github.com/Nassim-sadi/Go-Image-handler/releases).

### Build from Source

Requirements:
- Go 1.21+
- GCC (MinGW on Windows, Xcode on macOS, GCC on Linux)

```bash
# Clone the repository
git clone https://github.com/Nassim-sadi/Go-Image-handler.git
cd Go-Image-handler

# Build
go build -o imagehandler.exe

# Run
./imagehandler.exe
```

## Usage

1. **Create a Project** - Click "+ New Project" to create a project with an output folder
2. **Create a Preset** - Set width, height, format, quality, and resize mode
3. **Add Images** - Paste URLs or add local files
4. **Preview** - Select an image to preview with current settings
5. **Process** - Click "Process All" to batch process

### Color Adjustments

- **Saturation**: -100 to +100 (negative = desaturate, positive = saturate)
- **Brightness**: -100 to +100
- **Contrast**: -100 to +100

### Resize Modes

- **Fit**: Scale to fit within bounds (maintains aspect ratio)
- **Exact**: Force exact dimensions (may stretch)
- **Cover**: Fill bounds and crop (maintains aspect ratio)

## Keyboard Shortcuts

- `Ctrl+V` - Paste URLs from clipboard

## Configuration

Projects and presets are saved to:
- Windows: `%APPDATA%\ImageHandler\config.json`
- macOS/Linux: `~/.imagehandler/config.json`

## Tech Stack

- **Go** - Fast, compiled language
- **Fyne** - Cross-platform GUI framework
- **chai2010/webp** - WebP encoding
- **golang.org/x/image** - Image processing

## License

MIT License - see [LICENSE](LICENSE) file.

## Contributing

Contributions welcome! Please open an issue or pull request.
