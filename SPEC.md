# ImageHandler - Personal Image Downloader & Processor

## Concept & Vision
A fast, no-nonsense desktop tool for downloading images from URLs and batch processing them with saved presets. Built for designers/developers who repeatedly resize and convert images to WebP. Minimal UI, maximum speed - Go-powered for near-instant processing.

## Design Language
- **Aesthetic**: Dark, utilitarian interface - like a power tool, not a toy
- **Colors**: 
  - Background: `#1a1a2e` (deep navy)
  - Surface: `#16213e` (darker panel)
  - Accent: `#e94560` (vibrant red-pink)
  - Text: `#eaeaea` (off-white)
  - Success: `#4ecca3` (mint green)
- **Typography**: System monospace for data, sans-serif for labels
- **Motion**: Minimal - instant feedback, no fancy animations

## Features

### 1. Project Management
- Create/rename/delete projects
- Each project stores: name, default preset, output folder
- Double-click to select active project
- Projects stored in `~/.imagehandler/projects.json`

### 2. Preset System
- Presets contain: width, height, format (WebP/JPEG/PNG), quality (1-100)
- Resize modes:
  - **Exact**: Force exact dimensions (may stretch)
  - **Fit**: Fit within bounds (maintain aspect)
  - **Cover**: Fill bounds (may crop)
- Multiple presets per project
- Default preset auto-selected on project switch

### 3. Live Color Adjustments
- Saturation slider (-100 to +100)
- Brightness slider (-100 to +100)  
- Contrast slider (-100 to +100)
- Live preview of adjustments
- Applied during processing

### 4. URL/File Queue
- Paste multiple URLs (one per line)
- Add local files via file picker
- Drag & drop files/URLs onto window
- Clipboard paste (Ctrl+V)
- Visual indicators: pending (gray), downloading (blue), processing (yellow), done (green), error (red)
- Click to remove individual items
- Duplicate URL detection
- Batch operations: Process All, Clear Queue, Cancel

### 5. Download & Processing
- Async download with progress
- Process immediately after download
- Skip duplicates (by URL hash)
- Error handling: retry once on failure, then mark error
- Support formats: JPEG, PNG, GIF, BMP, TIFF, WebP
- 5 concurrent workers (configurable in future)

### 6. Image Preview
- Before/after preview panel
- Shows original vs processed
- Zoom controls
- Quick preview of color adjustments

### 7. Output
- Organized in project folder
- Naming: `{original_name}_{preset_name}_{timestamp}.{format}`
- Summary notification after batch complete

### 8. Progress & Feedback
- Progress bar for batch operations
- Cancel button to stop processing
- Real-time status updates
- Completion notification with output folder

## Technical Approach

### Stack
- **Language**: Go 1.21+
- **GUI**: Fyne 2.4+ (cross-platform, pure Go)
- **Image Processing**: `golang.org/x/image` + `github.com/chai2010/webp`
- **HTTP**: Standard library `net/http`
- **Storage**: JSON files in app config directory

### Performance Targets
- Download: 5 concurrent downloads
- Process 100 images: < 5 seconds (depends on image size)
- App startup: < 1 second
- Memory: < 200MB typical usage
