package processor

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/chai2010/webp"
	ximage "golang.org/x/image/draw"
	"imagehandler/models"
)

type ImageProcessor struct {
	useCWebP bool
}

func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{
		useCWebP: checkCWebP(),
	}
}

func checkCWebP() bool {
	cmd := exec.Command("cwebp", "-version")
	if err := cmd.Run(); err == nil {
		return true
	}
	if runtime.GOOS == "windows" {
		cmd := exec.Command("where", "cwebp")
		if err := cmd.Run(); err == nil {
			return true
		}
	}
	return false
}

func (p *ImageProcessor) ensureCWebP() error {
	if p.useCWebP {
		return nil
	}

	if runtime.GOOS != "windows" {
		return fmt.Errorf("cwebp not available for this OS")
	}

	cwebpPath := filepath.Join(os.TempDir(), "imagehandler_cwebp.exe")
	if _, err := os.Stat(cwebpPath); err == nil {
		os.Setenv("PATH", filepath.Dir(cwebpPath)+string(os.PathListSeparator)+os.Getenv("PATH"))
		p.useCWebP = true
		return nil
	}

	url := "https://storage.googleapis.com/downloads.webmproject.org/releases/webp/cwebp-windows-x64-1.3.2.zip"
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download cwebp: %w", err)
	}
	defer resp.Body.Close()

	zipFile := filepath.Join(os.TempDir(), "cwebp.zip")
	out, err := os.Create(zipFile)
	if err != nil {
		return err
	}
	io.Copy(out, resp.Body)
	out.Close()

	cmd := exec.Command("tar", "-xf", zipFile, "-C", os.TempDir())
	cmd.Run()
	os.Remove(zipFile)

	dllPath := filepath.Join(os.TempDir(), "cwebp-windows-x64-1.3.2", "bin", "cwebp.exe")
	if _, err := os.Stat(dllPath); err == nil {
		os.Rename(dllPath, cwebpPath)
		os.Setenv("PATH", filepath.Dir(cwebpPath)+string(os.PathListSeparator)+os.Getenv("PATH"))
		p.useCWebP = true
	}
	return nil
}

func (p *ImageProcessor) ProcessImage(inputPath string, preset *models.Preset, outputDir string) (string, error) {
	srcFile, err := os.Open(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to open image: %w", err)
	}
	defer srcFile.Close()

	srcImg, _, err := image.Decode(srcFile)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := srcImg.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	resizedImg := p.resize(srcImg, srcWidth, srcHeight, preset)

	adjustedImg := p.applyColorAdjustments(resizedImg, preset)

	baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	outputExt := p.getOutputExtension(preset.Format)
	timestamp := fmt.Sprintf("%d", os.Getpid())
	outputName := fmt.Sprintf("%s_%s_%s%s", baseName, preset.Name, timestamp, outputExt)
	outputPath := filepath.Join(outputDir, outputName)

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	switch preset.Format {
	case "webp":
		err = p.saveWebP(adjustedImg, outputPath, preset.Quality)
	case "jpeg", "jpg":
		err = p.saveJPEG(adjustedImg, outputPath, preset.Quality)
	case "png":
		err = p.savePNG(adjustedImg, outputPath)
	default:
		err = p.savePNG(adjustedImg, outputPath)
	}

	if err != nil {
		return "", fmt.Errorf("failed to save image: %w", err)
	}

	return outputPath, nil
}

func (p *ImageProcessor) PreviewImage(inputPath string, preset *models.Preset, maxSize int) (image.Image, error) {
	srcFile, err := os.Open(inputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer srcFile.Close()

	srcImg, _, err := image.Decode(srcFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := srcImg.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	scale := float64(maxSize) / float64(srcWidth)
	if float64(srcHeight)*scale > float64(maxSize) {
		scale = float64(maxSize) / float64(srcHeight)
	}

	if scale < 1 {
		previewW := int(float64(srcWidth) * scale)
		previewH := int(float64(srcHeight) * scale)
		previewImg := image.NewRGBA(image.Rect(0, 0, previewW, previewH))
		catmullRom := ximage.CatmullRom
		catmullRom.Scale(previewImg, previewImg.Bounds(), srcImg, srcImg.Bounds(), ximage.Over, nil)
		srcImg = previewImg
	}

	adjustedImg := p.applyColorAdjustments(srcImg, preset)

	return adjustedImg, nil
}

func (p *ImageProcessor) applyColorAdjustments(img image.Image, preset *models.Preset) image.Image {
	if preset.Saturation == 0 && preset.Brightness == 0 && preset.Contrast == 0 {
		return img
	}

	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			rf := float64(r>>8) / 255.0
			gf := float64(g>>8) / 255.0
			bf := float64(b>>8) / 255.0

			if preset.Saturation != 0 {
				gray := 0.299*rf + 0.587*gf + 0.114*bf
				factor := 1.0 + preset.Saturation/100.0
				rf = gray + factor*(rf-gray)
				gf = gray + factor*(gf-gray)
				bf = gray + factor*(bf-gray)
			}

			if preset.Brightness != 0 {
				brightness := preset.Brightness / 100.0
				rf += brightness
				gf += brightness
				bf += brightness
			}

			if preset.Contrast != 0 {
				contrast := (100.0 + preset.Contrast) / 100.0
				contrast = contrast * contrast
				rf = ((rf - 0.5) * contrast) + 0.5
				gf = ((gf - 0.5) * contrast) + 0.5
				bf = ((bf - 0.5) * contrast) + 0.5
			}

			if rf < 0 {
				rf = 0
			}
			if rf > 1 {
				rf = 1
			}
			if gf < 0 {
				gf = 0
			}
			if gf > 1 {
				gf = 1
			}
			if bf < 0 {
				bf = 0
			}
			if bf > 1 {
				bf = 1
			}

			newImg.Set(x, y, color.RGBA{
				R: uint8(rf * 255),
				G: uint8(gf * 255),
				B: uint8(bf * 255),
				A: uint8(a >> 8),
			})
		}
	}

	return newImg
}

func (p *ImageProcessor) resize(srcImg image.Image, srcW, srcH int, preset *models.Preset) image.Image {
	targetW := preset.Width
	targetH := preset.Height

	if targetW == 0 && targetH == 0 {
		return srcImg
	}

	var newW, newH int

	switch preset.Mode {
	case "exact":
		if targetW == 0 {
			targetW = srcW
		}
		if targetH == 0 {
			targetH = srcH
		}
		newW, newH = targetW, targetH

	case "fit":
		if targetW == 0 {
			ratio := float64(targetH) / float64(srcH)
			newW = int(float64(srcW) * ratio)
			newH = targetH
		} else if targetH == 0 {
			ratio := float64(targetW) / float64(srcW)
			newW = targetW
			newH = int(float64(srcH) * ratio)
		} else {
			ratioW := float64(targetW) / float64(srcW)
			ratioH := float64(targetH) / float64(srcH)
			ratio := ratioW
			if ratioH < ratio {
				ratio = ratioH
			}
			newW = int(float64(srcW) * ratio)
			newH = int(float64(srcH) * ratio)
		}

	case "cover":
		ratioW := float64(targetW) / float64(srcW)
		ratioH := float64(targetH) / float64(srcH)
		ratio := ratioW
		if ratioH > ratio {
			ratio = ratioH
		}
		newW = int(float64(srcW) * ratio)
		newH = int(float64(srcH) * ratio)
		if newW < targetW {
			newW = targetW
		}
		if newH < targetH {
			newH = targetH
		}

	default:
		if targetW == 0 {
			ratio := float64(targetH) / float64(srcH)
			newW = int(float64(srcW) * ratio)
			newH = targetH
		} else if targetH == 0 {
			ratio := float64(targetW) / float64(srcW)
			newW = targetW
			newH = int(float64(srcH) * ratio)
		} else {
			ratioW := float64(targetW) / float64(srcW)
			ratioH := float64(targetH) / float64(srcH)
			ratio := ratioW
			if ratioH < ratio {
				ratio = ratioH
			}
			newW = int(float64(srcW) * ratio)
			newH = int(float64(srcH) * ratio)
		}
	}

	if newW <= 0 {
		newW = 1
	}
	if newH <= 0 {
		newH = 1
	}

	dstImg := image.NewRGBA(image.Rect(0, 0, newW, newH))

	catmullRom := ximage.CatmullRom
	catmullRom.Scale(dstImg, dstImg.Bounds(), srcImg, srcImg.Bounds(), ximage.Over, nil)

	if preset.Mode == "cover" && (newW > targetW || newH > targetH) {
		offsetX := (newW - targetW) / 2
		offsetY := (newH - targetH) / 2

		cropped := image.NewRGBA(image.Rect(0, 0, targetW, targetH))

		for y := 0; y < targetH; y++ {
			for x := 0; x < targetW; x++ {
				srcX := x + offsetX
				srcY := y + offsetY
				if srcX >= 0 && srcX < newW && srcY >= 0 && srcY < newH {
					cropped.Set(x, y, dstImg.At(srcX, srcY))
				}
			}
		}

		return cropped
	}

	return dstImg
}

func (p *ImageProcessor) saveWebP(img image.Image, path string, quality int) error {
	bounds := img.Bounds()
	var rgba *image.RGBA
	if nrgba, ok := img.(*image.RGBA); ok {
		rgba = nrgba
	} else {
		rgba = image.NewRGBA(bounds)
		draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	}

	if !p.useCWebP {
		p.ensureCWebP()
	}

	if p.useCWebP {
		return p.saveWebPCWebP(rgba, path, quality)
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	options := &webp.Options{
		Quality: float32(quality),
	}

	if quality >= 95 {
		options.Lossless = true
	}

	return webp.Encode(file, rgba, options)
}

func (p *ImageProcessor) saveWebPCWebP(img *image.RGBA, outputPath string, quality int) error {
	tmpFile, err := os.CreateTemp("", "webp_*.png")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	if err := png.Encode(tmpFile, img); err != nil {
		os.Remove(tmpPath)
		return err
	}

	var cwebpArgs []string
	if quality >= 95 {
		cwebpArgs = []string{tmpPath, "-lossless", "-o", outputPath}
	} else {
		cwebpArgs = []string{tmpPath, "-q", fmt.Sprintf("%d", quality), "-o", outputPath}
	}

	cmd := exec.Command("cwebp", cwebpArgs...)
	if err := cmd.Run(); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("cwebp failed: %w", err)
	}

	os.Remove(tmpPath)
	return nil
}

func (p *ImageProcessor) saveJPEG(img image.Image, path string, quality int) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var rgbaImg image.Image = img
	if _, ok := img.(*image.RGBA); !ok {
		bounds := img.Bounds()
		rgba := image.NewRGBA(bounds)
		draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
		rgbaImg = rgba
	}

	options := &jpeg.Options{
		Quality: quality,
	}

	return jpeg.Encode(file, rgbaImg, options)
}

func (p *ImageProcessor) savePNG(img image.Image, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	bounds := img.Bounds()
	var rgbaImg image.Image = img
	if _, ok := img.(*image.RGBA); !ok {
		rgba := image.NewRGBA(bounds)
		draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
		rgbaImg = rgba
	}

	return png.Encode(file, rgbaImg)
}

func (p *ImageProcessor) getOutputExtension(format string) string {
	switch format {
	case "webp":
		return ".webp"
	case "jpeg", "jpg":
		return ".jpg"
	case "png":
		return ".png"
	default:
		return ".webp"
	}
}
