package gui

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"imagehandler/config"
	"imagehandler/models"
	"imagehandler/processor"
)

type App struct {
	fyneApp fyne.App
	window  fyne.Window

	projectsList *widget.List
	presetsList  *widget.List
	queueList    *widget.List

	projectBinding binding.StringList
	presetBinding  binding.StringList

	currentProject *models.Project
	currentPreset  *models.Preset

	widthEntry      *widget.Entry
	heightEntry     *widget.Entry
	formatSelect    *widget.Select
	qualitySlider   *widget.Slider
	modeSelect      *widget.Select
	outputEntry     *widget.Entry
	presetNameEntry *widget.Entry

	saturationSlider *widget.Slider
	brightnessSlider *widget.Slider
	contrastSlider   *widget.Slider

	urlEntry *widget.Entry

	queueItems []*models.QueueItem
	queueMutex sync.Mutex

	downloader     *processor.Downloader
	imageProcessor *processor.ImageProcessor

	statusLabel  *canvas.Text
	progressBar  *widget.ProgressBar
	cancelButton *widget.Button
	cancelChan   chan struct{}
	processing   bool

	previewImage       *canvas.Image
	previewMutex       sync.RWMutex
	outputLabel        *widget.Label
	currentPreviewPath string
}

const (
	colorBg      = "#1a1a2e"
	colorSurface = "#16213e"
	colorAccent  = "#e94560"
	colorText    = "#eaeaea"
	colorSuccess = "#4ecca3"
)

func (a *App) Run() {
	a.fyneApp = app.New()
	a.window = a.fyneApp.NewWindow("ImageHandler")

	a.downloader = processor.NewDownloader(5)
	a.imageProcessor = processor.NewImageProcessor()

	if err := config.Load(); err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
	}

	a.setupUI()
	a.setupShortcuts()
	a.setupDragDrop()
	a.loadProjects()

	a.window.Resize(fyne.NewSize(1000, 700))
	a.window.Show()
	a.fyneApp.Run()
}

func (a *App) setupUI() {
	a.projectBinding = binding.NewStringList()
	a.presetBinding = binding.NewStringList()

	projectsLabel := widget.NewLabel("PROJECTS")
	projectsLabel.TextStyle = fyne.TextStyle{Bold: true}

	a.projectsList = widget.NewList(
		func() int { return a.projectBinding.Length() },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.TextStyle = fyne.TextStyle{Monospace: true}
			return label
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			items, _ := a.projectBinding.Get()
			if id < len(items) {
				obj.(*widget.Label).SetText(items[id])
			}
		},
	)

	a.projectsList.OnSelected = func(id widget.ListItemID) {
		a.onProjectSelected(id)
	}

	newProjectBtn := widget.NewButton("+ New Project", a.onNewProject)
	deleteProjectBtn := widget.NewButton("Delete", a.onDeleteProject)

	projectsPanel := container.NewVBox(
		projectsLabel,
		container.NewScroll(a.projectsList),
		container.NewHBox(newProjectBtn, deleteProjectBtn),
	)

	presetLabel := widget.NewLabel("PRESET")
	presetLabel.TextStyle = fyne.TextStyle{Bold: true}

	a.presetsList = widget.NewList(
		func() int { return a.presetBinding.Length() },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.TextStyle = fyne.TextStyle{Monospace: true}
			return label
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			items, _ := a.presetBinding.Get()
			if id < len(items) {
				obj.(*widget.Label).SetText(items[id])
			}
		},
	)

	a.presetsList.OnSelected = func(id widget.ListItemID) {
		a.onPresetSelected(id)
	}

	a.presetNameEntry = widget.NewEntry()
	a.presetNameEntry.SetPlaceHolder("Preset name...")

	widthLabel := widget.NewLabel("Width:")
	a.widthEntry = widget.NewEntry()
	a.widthEntry.SetPlaceHolder("800")

	heightLabel := widget.NewLabel("Height:")
	a.heightEntry = widget.NewEntry()
	a.heightEntry.SetPlaceHolder("600")

	formatLabel := widget.NewLabel("Format:")
	a.formatSelect = widget.NewSelect([]string{"webp", "jpeg", "png"}, func(s string) {})
	a.formatSelect.SetSelected("webp")

	qualityLabel := widget.NewLabel("Quality:")
	a.qualitySlider = widget.NewSlider(1, 100)
	a.qualitySlider.SetValue(85)

	modeLabel := widget.NewLabel("Mode:")
	a.modeSelect = widget.NewSelect([]string{"fit", "exact", "cover"}, func(s string) {})
	a.modeSelect.SetSelected("fit")

	outputLabel := widget.NewLabel("Output:")
	a.outputEntry = widget.NewEntry()
	a.outputEntry.SetPlaceHolder("Select output folder...")
	browseBtn := widget.NewButton("Browse", a.onBrowseOutput)

	savePresetBtn := widget.NewButton("Save Preset", a.onSavePreset)
	newPresetBtn := widget.NewButton("+ New", a.onNewPreset)
	deletePresetBtn := widget.NewButton("Delete", a.onDeletePreset)

	outputRow := container.NewBorder(nil, nil, outputLabel, browseBtn, a.outputEntry)

	satLabel := widget.NewLabel("Saturation:")
	a.saturationSlider = widget.NewSlider(-100, 100)
	a.saturationSlider.SetValue(0)
	a.saturationSlider.OnChanged = func(float64) { a.updatePreview() }

	brightLabel := widget.NewLabel("Brightness:")
	a.brightnessSlider = widget.NewSlider(-100, 100)
	a.brightnessSlider.SetValue(0)
	a.brightnessSlider.OnChanged = func(float64) { a.updatePreview() }

	contrastLabel := widget.NewLabel("Contrast:")
	a.contrastSlider = widget.NewSlider(-100, 100)
	a.contrastSlider.SetValue(0)
	a.contrastSlider.OnChanged = func(float64) { a.updatePreview() }

	resetColorBtn := widget.NewButton("Reset Color", a.onResetColor)

	presetPanel := container.NewVBox(
		presetLabel,
		a.presetNameEntry,
		container.NewGridWithColumns(2, widthLabel, a.widthEntry, heightLabel, a.heightEntry),
		container.NewGridWithColumns(2, formatLabel, a.formatSelect),
		container.NewGridWithColumns(2, qualityLabel, a.qualitySlider),
		container.NewGridWithColumns(2, modeLabel, a.modeSelect),
		outputRow,
		container.NewHBox(savePresetBtn, newPresetBtn, deletePresetBtn),
		widget.NewSeparator(),
		satLabel, a.saturationSlider,
		brightLabel, a.brightnessSlider,
		contrastLabel, a.contrastSlider,
		resetColorBtn,
		widget.NewSeparator(),
		container.NewScroll(a.presetsList),
	)

	previewLabel := widget.NewLabel("PREVIEW")
	previewLabel.TextStyle = fyne.TextStyle{Bold: true}

	a.previewImage = canvas.NewImageFromImage(nil)
	a.previewImage.FillMode = canvas.ImageFillContain
	a.previewImage.SetMinSize(fyne.NewSize(350, 300))

	previewPanel := container.NewVBox(
		previewLabel,
		a.previewImage,
	)

	urlLabel := widget.NewLabel("INPUT")
	urlLabel.TextStyle = fyne.TextStyle{Bold: true}

	a.urlEntry = widget.NewEntry()
	a.urlEntry.SetPlaceHolder("Paste image URL here...")

	addUrlsBtn := widget.NewButton("Add URL", a.onAddUrls)
	addFilesBtn := widget.NewButton("Add Files", a.onAddFiles)
	clearBtn := widget.NewButton("Clear", a.onClearQueue)

	inputRow := container.NewVBox(
		a.urlEntry,
		container.NewHBox(addUrlsBtn, addFilesBtn),
	)

	a.queueList = widget.NewList(
		func() int { return len(a.queueItems) },
		func() fyne.CanvasObject {
			label := widget.NewLabel("")
			label.Wrapping = fyne.TextWrapWord
			return label
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < len(a.queueItems) {
				item := a.queueItems[id]
				status := ""
				switch item.Status {
				case "downloading":
					status = " [DL]"
				case "processing":
					status = " [PROC]"
				case "done":
					status = " [OK]"
				case "error":
					status = " [ERR]"
				default:
					status = " [---]"
				}
				displayName := filepath.Base(item.URL)
				if displayName == "." || displayName == "" {
					displayName = item.URL
					if len(displayName) > 50 {
						displayName = "..." + displayName[len(displayName)-50:]
					}
				}
				obj.(*widget.Label).SetText(displayName + status)
			}
		},
	)

	a.queueList.OnSelected = func(id widget.ListItemID) {
		if id < len(a.queueItems) {
			item := a.queueItems[id]
			if !strings.HasPrefix(item.URL, "http://") && !strings.HasPrefix(item.URL, "https://") {
				a.urlEntry.SetText(item.URL)
				a.updatePreviewForItem(item)
			}
		}
	}

	a.queueList.OnUnselected = func(id widget.ListItemID) {
		if id < len(a.queueItems) {
			a.queueItems = append(a.queueItems[:id], a.queueItems[id+1:]...)
			a.queueList.Refresh()
		}
	}

	queueLabel := widget.NewLabel("QUEUE")
	queueLabel.TextStyle = fyne.TextStyle{Bold: true}

	a.progressBar = widget.NewProgressBar()
	a.progressBar.Hide()

	a.cancelButton = widget.NewButton("Cancel", a.onCancel)
	a.cancelButton.Hide()

	processBtn := widget.NewButton("Process All", a.onProcessAll)
	showImageBtn := widget.NewButton("Show Image", a.onShowImage)

	a.outputLabel = widget.NewLabel("Output: -")
	a.outputLabel.TextStyle = fyne.TextStyle{Monospace: true}

	queueScroll := container.NewScroll(a.queueList)
	queueScroll.SetMinSize(fyne.NewSize(0, 200))

	queuePanel := container.NewVBox(
		urlLabel,
		inputRow,
		queueLabel,
		queueScroll,
		container.NewHBox(clearBtn, showImageBtn),
		a.progressBar,
		container.NewHBox(processBtn, a.cancelButton),
		a.outputLabel,
	)

	leftPanel := container.NewVBox(projectsPanel)
	centerPanel := container.NewVBox(presetPanel)
	rightPanel := container.NewVBox(previewPanel, queuePanel)

	mainContainer := container.NewHSplit(
		container.NewHSplit(leftPanel, centerPanel),
		rightPanel,
	)
	mainContainer.SetOffset(0.33)

	a.statusLabel = canvas.NewText("Ready", theme.Color(theme.ColorNameForeground))
	a.statusLabel.TextStyle = fyne.TextStyle{Monospace: true}

	statusBar := container.NewBorder(nil, nil, nil, a.statusLabel)

	content := container.NewBorder(nil, statusBar, nil, nil, mainContainer)

	a.window.SetContent(content)
}

func (a *App) setupShortcuts() {
}

func (a *App) setupDragDrop() {
}

func (a *App) loadProjects() {
	projects := config.GetAllProjects()
	names := make([]string, len(projects))
	for i, p := range projects {
		names[i] = p.Name
	}
	a.projectBinding.Set(names)
}

func (a *App) loadPresets() {
	if a.currentProject == nil {
		a.presetBinding.Set([]string{})
		return
	}
	presets := config.GetProjectPresets(a.currentProject.ID)
	names := make([]string, len(presets))
	for i, p := range presets {
		names[i] = p.Name
	}
	a.presetBinding.Set(names)
}

func (a *App) onProjectSelected(id widget.ListItemID) {
	projects := config.GetAllProjects()
	if id < len(projects) {
		a.currentProject = projects[id]
		a.outputEntry.SetText(a.currentProject.OutputPath)
		a.outputLabel.SetText("Output: " + a.currentProject.OutputPath)
		a.loadPresets()

		if len(a.currentProject.PresetIDs) > 0 {
			a.presetsList.Select(0)
		}
	}
}

func (a *App) onPresetSelected(id widget.ListItemID) {
	if a.currentProject == nil {
		return
	}
	presets := config.GetProjectPresets(a.currentProject.ID)
	if id < len(presets) {
		a.currentPreset = presets[id]
		a.loadPresetValues()
		a.updatePreview()
	}
}

func (a *App) loadPresetValues() {
	if a.currentPreset == nil {
		return
	}
	a.presetNameEntry.SetText(a.currentPreset.Name)
	a.widthEntry.SetText(fmt.Sprintf("%d", a.currentPreset.Width))
	a.heightEntry.SetText(fmt.Sprintf("%d", a.currentPreset.Height))
	a.formatSelect.SetSelected(a.currentPreset.Format)
	a.qualitySlider.SetValue(float64(a.currentPreset.Quality))
	a.modeSelect.SetSelected(a.currentPreset.Mode)
	a.saturationSlider.SetValue(a.currentPreset.Saturation)
	a.brightnessSlider.SetValue(a.currentPreset.Brightness)
	a.contrastSlider.SetValue(a.currentPreset.Contrast)
}

func (a *App) onNewProject() {
	projectName := fmt.Sprintf("Project %d", len(config.GetAllProjects())+1)
	defaultPath := filepath.Join(os.Getenv("USERPROFILE"), "Pictures", projectName)

	project := models.NewProject(projectName, defaultPath)
	config.AddProject(project)

	a.loadProjects()

	for i, p := range config.GetAllProjects() {
		if p.ID == project.ID {
			a.projectsList.Select(i)
			break
		}
	}

	a.statusLabel.Text = fmt.Sprintf("Created project: %s", projectName)
}

func (a *App) onDeleteProject() {
	if a.currentProject == nil {
		return
	}
	config.DeleteProject(a.currentProject.ID)
	a.currentProject = nil
	a.loadProjects()
	a.loadPresets()
	a.statusLabel.Text = "Project deleted"
}

func (a *App) onBrowseOutput() {
	fdialog := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}
		a.outputEntry.SetText(uri.Path())
		if a.currentProject != nil {
			a.currentProject.OutputPath = uri.Path()
			config.Save()
		}
	}, a.window)
	fdialog.Show()
}

func (a *App) onNewPreset() {
	if a.currentProject == nil {
		a.statusLabel.Text = "Select a project first"
		return
	}

	preset := models.NewPreset("New Preset", 800, 600, 85, "webp", "fit")
	config.AddPreset(preset)
	config.AddPresetToProject(a.currentProject.ID, preset.ID)

	a.loadPresets()

	for i, p := range config.GetProjectPresets(a.currentProject.ID) {
		if p.ID == preset.ID {
			a.presetsList.Select(i)
			break
		}
	}
}

func (a *App) onDeletePreset() {
	if a.currentPreset == nil || a.currentProject == nil {
		return
	}
	config.RemovePresetFromProject(a.currentProject.ID, a.currentPreset.ID)
	config.DeletePreset(a.currentPreset.ID)
	a.currentPreset = nil
	a.loadPresets()
	a.statusLabel.Text = "Preset deleted"
}

func (a *App) onSavePreset() {
	var width, height int
	fmt.Sscanf(a.widthEntry.Text, "%d", &width)
	fmt.Sscanf(a.heightEntry.Text, "%d", &height)

	presetName := a.presetNameEntry.Text
	if presetName == "" {
		presetName = fmt.Sprintf("Preset_%dx%d", width, height)
	}

	if a.currentPreset == nil {
		if a.currentProject == nil {
			a.statusLabel.Text = "Create a project first"
			return
		}
		a.currentPreset = models.NewPreset(presetName, width, height, 85, "webp", "fit")
		config.AddPreset(a.currentPreset)
		config.AddPresetToProject(a.currentProject.ID, a.currentPreset.ID)
	} else {
		a.currentPreset.Name = presetName
		a.currentPreset.Width = width
		a.currentPreset.Height = height
		a.currentPreset.Format = a.formatSelect.Selected
		a.currentPreset.Quality = int(a.qualitySlider.Value)
		a.currentPreset.Mode = a.modeSelect.Selected
	}

	a.currentPreset.Saturation = a.saturationSlider.Value
	a.currentPreset.Brightness = a.brightnessSlider.Value
	a.currentPreset.Contrast = a.contrastSlider.Value

	config.Save()
	a.loadPresets()

	a.statusLabel.Text = fmt.Sprintf("Saved preset: %s", presetName)
}

func (a *App) onResetColor() {
	a.saturationSlider.SetValue(0)
	a.brightnessSlider.SetValue(0)
	a.contrastSlider.SetValue(0)
	a.updatePreview()
}

func (a *App) onAddUrls() {
	text := a.urlEntry.Text
	if text == "" {
		return
	}

	lines := a.parseURLs(text)
	if len(lines) == 0 {
		return
	}

	count := 0
	for _, url := range lines {
		item := models.NewQueueItem(url, "", "")
		if a.currentProject != nil {
			item.ProjectID = a.currentProject.ID
		}
		if a.currentPreset != nil {
			item.PresetID = a.currentPreset.ID
		}
		a.queueMutex.Lock()
		a.queueItems = append(a.queueItems, item)
		a.queueMutex.Unlock()
		count++
	}

	a.urlEntry.SetText("")
	a.updateQueueDisplay()

	a.statusLabel.Text = fmt.Sprintf("Added %d URLs to queue", count)
}

func (a *App) onPaste() {
	cb := fyne.CurrentApp().Clipboard()

	text := cb.Content()
	if text == "" {
		return
	}

	if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
		a.urlEntry.SetText(text)
		a.onAddUrls()
	} else {
		lines := a.parseURLs(text)
		if len(lines) > 0 {
			a.urlEntry.SetText(text)
			a.onAddUrls()
		}
	}
}

func (a *App) onAddFiles() {
	fdialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()

		uri := reader.URI()
		if uri == nil {
			return
		}

		uriStr := uri.Path()
		stat, err := os.Stat(uriStr)
		if err != nil {
			return
		}

		if stat.IsDir() {
			files, _ := os.ReadDir(uriStr)
			a.queueMutex.Lock()
			count := 0
			for _, f := range files {
				if f.IsDir() {
					continue
				}
				ext := strings.ToLower(filepath.Ext(f.Name()))
				if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" || ext == ".bmp" || ext == ".webp" || ext == ".tiff" || ext == ".tif" {
					filePath := filepath.Join(uriStr, f.Name())
					a.addFileToQueue(filePath)
					count++
				}
			}
			a.queueMutex.Unlock()
			a.updateQueueDisplay()
			a.statusLabel.Text = fmt.Sprintf("Added %d files to queue", count)
		} else {
			a.addFileToQueue(uriStr)
			a.updateQueueDisplay()
		}
	}, a.window)
	fdialog.Show()
}

func (a *App) addFileToQueue(filePath string) {
	item := models.NewQueueItem(filePath, "", "")
	item.FileName = filepath.Base(filePath)
	if a.currentProject != nil {
		item.ProjectID = a.currentProject.ID
	}
	if a.currentPreset != nil {
		item.PresetID = a.currentPreset.ID
	}
	a.queueMutex.Lock()
	a.queueItems = append(a.queueItems, item)
	a.queueMutex.Unlock()
}

func (a *App) addURLToQueue(url string) {
	item := models.NewQueueItem(url, "", "")
	if a.currentProject != nil {
		item.ProjectID = a.currentProject.ID
	}
	if a.currentPreset != nil {
		item.PresetID = a.currentPreset.ID
	}
	a.queueMutex.Lock()
	a.queueItems = append(a.queueItems, item)
	a.queueMutex.Unlock()
}

func (a *App) parseURLs(text string) []string {
	var urls []string
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && (strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://")) {
			urls = append(urls, line)
		}
	}

	return urls
}

func (a *App) updateQueueDisplay() {
	if a.queueList != nil {
		a.queueList.Refresh()
	}
}

func (a *App) onClearQueue() {
	a.queueMutex.Lock()
	a.queueItems = a.queueItems[:0]
	a.queueMutex.Unlock()
	if a.queueList != nil {
		a.queueList.Refresh()
	}
	a.statusLabel.Text = "Queue cleared"
}

func (a *App) onCancel() {
	if a.cancelChan != nil {
		close(a.cancelChan)
	}
	a.cancelChan = nil
	a.processing = false
	a.progressBar.Hide()
	a.cancelButton.Hide()
	a.statusLabel.Text = "Cancelled"
}

func (a *App) onProcessAll() {
	if len(a.queueItems) == 0 {
		a.statusLabel.Text = "Queue is empty"
		return
	}

	if a.currentProject == nil {
		a.statusLabel.Text = "Select a project first"
		return
	}

	if a.currentPreset == nil {
		a.statusLabel.Text = "Select a preset first"
		return
	}

	a.syncPresetFromUI()

	a.processing = true
	a.cancelChan = make(chan struct{})
	a.progressBar.Show()
	a.progressBar.SetValue(0)
	a.cancelButton.Show()
	a.processQueue()
}

func (a *App) syncPresetFromUI() {
	if a.currentPreset == nil {
		return
	}

	var width, height int
	fmt.Sscanf(a.widthEntry.Text, "%d", &width)
	fmt.Sscanf(a.heightEntry.Text, "%d", &height)

	presetName := a.presetNameEntry.Text
	if presetName == "" {
		presetName = fmt.Sprintf("Preset_%dx%d", width, height)
	}

	a.currentPreset.Name = presetName
	a.currentPreset.Width = width
	a.currentPreset.Height = height
	a.currentPreset.Format = a.formatSelect.Selected
	a.currentPreset.Quality = int(a.qualitySlider.Value)
	a.currentPreset.Mode = a.modeSelect.Selected
	a.currentPreset.Saturation = a.saturationSlider.Value
	a.currentPreset.Brightness = a.brightnessSlider.Value
	a.currentPreset.Contrast = a.contrastSlider.Value
}

func (a *App) processQueue() {
	pending := 0
	for _, item := range a.queueItems {
		if item.Status == "pending" {
			pending++
		}
	}

	if pending == 0 {
		a.statusLabel.Text = "No pending items"
		a.progressBar.Hide()
		a.cancelButton.Hide()
		return
	}

	total := pending
	completed := 0
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5)

	for _, item := range a.queueItems {
		if item.Status != "pending" {
			continue
		}

		select {
		case <-a.cancelChan:
			a.processing = false
			a.progressBar.Hide()
			a.cancelButton.Hide()
			a.statusLabel.Text = fmt.Sprintf("Cancelled (%d/%d done)", completed, total)
			return
		default:
		}

		semaphore <- struct{}{}
		wg.Add(1)

		go func(qi *models.QueueItem) {
			defer func() {
				<-semaphore
				wg.Done()
			}()

			a.processItem(qi)
			completed++

			fyne.Do(func() {
				a.progressBar.SetValue(float64(completed) / float64(total))
				a.statusLabel.Text = fmt.Sprintf("Processing: %d/%d", completed, total)
				a.queueList.Refresh()
			})
		}(item)
	}

	wg.Wait()

	fyne.Do(func() {
		a.processing = false
		a.progressBar.Hide()
		a.cancelButton.Hide()
		a.queueList.Refresh()
		a.statusLabel.Text = fmt.Sprintf("Completed %d images", completed)
		a.outputLabel.SetText("Output: " + a.currentProject.OutputPath)

		folder := a.currentProject.OutputPath
		dialog.ShowInformation("Done!", fmt.Sprintf("Processed %d image(s).\nOutput: %s", completed, folder), a.window)
	})
}

func (a *App) processItem(item *models.QueueItem) {
	project := a.currentProject
	preset := a.currentPreset

	if project == nil || preset == nil {
		item.Status = "error"
		item.Error = "No project or preset selected"
		return
	}

	var inputPath string

	if strings.HasPrefix(item.URL, "http://") || strings.HasPrefix(item.URL, "https://") {
		item.Status = "downloading"

		result, err := a.downloader.Download(item.URL)
		if err != nil {
			item.Status = "error"
			item.Error = err.Error()
			return
		}

		inputPath = result.FilePath
		defer os.Remove(result.FilePath)
	} else {
		if _, err := os.Stat(item.URL); os.IsNotExist(err) {
			item.Status = "error"
			item.Error = "File not found"
			return
		}
		inputPath = item.URL
	}

	item.Status = "processing"

	outputPath, err := a.imageProcessor.ProcessImage(inputPath, preset, project.OutputPath)
	if err != nil {
		item.Status = "error"
		item.Error = err.Error()
		return
	}

	item.Status = "done"
	item.FileName = filepath.Base(outputPath)
}

func (a *App) onShowImage() {
	if a.currentProject == nil || a.currentPreset == nil {
		a.statusLabel.Text = "Select project and preset first"
		return
	}

	fdialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()

		uri := reader.URI()
		if uri == nil {
			return
		}

		inputPath := uri.Path()
		stat, err := os.Stat(inputPath)
		if err != nil || stat.IsDir() {
			return
		}

		outputPath, err := a.imageProcessor.ProcessImage(inputPath, a.currentPreset, a.currentProject.OutputPath)
		if err != nil {
			dialog.ShowError(err, a.window)
			return
		}

		item := models.NewQueueItem(inputPath, a.currentProject.ID, a.currentPreset.ID)
		item.Status = "done"
		item.FileName = filepath.Base(outputPath)
		item.URL = inputPath

		a.queueMutex.Lock()
		a.queueItems = append(a.queueItems, item)
		a.queueMutex.Unlock()
		a.updateQueueDisplay()

		a.statusLabel.Text = "Done: " + item.FileName

		fyne.Do(func() {
			exec.Command("explorer", "/select,"+outputPath).Start()
		})
	}, a.window)
	fdialog.Show()
}

func (a *App) updatePreview() {
	if a.currentPreset == nil {
		a.currentPreset = &models.Preset{
			Width:      800,
			Height:     600,
			Format:     "webp",
			Quality:    85,
			Mode:       "fit",
			Saturation: a.saturationSlider.Value,
			Brightness: a.brightnessSlider.Value,
			Contrast:   a.contrastSlider.Value,
		}
	} else {
		a.currentPreset.Saturation = a.saturationSlider.Value
		a.currentPreset.Brightness = a.brightnessSlider.Value
		a.currentPreset.Contrast = a.contrastSlider.Value
	}

	if len(a.queueItems) == 0 {
		return
	}

	for _, item := range a.queueItems {
		if !strings.HasPrefix(item.URL, "http://") && !strings.HasPrefix(item.URL, "https://") {
			a.updatePreviewForItem(item)
			break
		}
	}
}

func (a *App) updatePreviewForItem(item *models.QueueItem) {
	if a.currentPreset == nil {
		return
	}

	a.currentPreset.Saturation = a.saturationSlider.Value
	a.currentPreset.Brightness = a.brightnessSlider.Value
	a.currentPreset.Contrast = a.contrastSlider.Value

	inputPath := item.URL
	if strings.HasPrefix(inputPath, "http://") || strings.HasPrefix(inputPath, "https://") {
		return
	}

	stat, err := os.Stat(inputPath)
	if err != nil || stat.IsDir() {
		return
	}

	go func() {
		preview, err := a.imageProcessor.PreviewImage(inputPath, a.currentPreset, 400)
		if err != nil {
			fyne.Do(func() {
				a.statusLabel.Text = "Preview error: " + err.Error()
			})
			return
		}

		fyne.Do(func() {
			a.previewMutex.Lock()
			a.previewImage.Image = preview
			a.previewMutex.Unlock()
			a.previewImage.Refresh()
		})
	}()
}

func (a *App) onSelectPreview() {
	fdialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil || reader == nil {
			return
		}
		defer reader.Close()

		uri := reader.URI()
		if uri == nil {
			return
		}

		inputPath := uri.Path()
		stat, err := os.Stat(inputPath)
		if err != nil || stat.IsDir() {
			return
		}

		ext := strings.ToLower(filepath.Ext(inputPath))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".bmp" && ext != ".webp" {
			dialog.ShowError(fmt.Errorf("unsupported format: %s", ext), a.window)
			return
		}

		a.previewMutex.Lock()
		a.currentPreviewPath = inputPath
		a.previewMutex.Unlock()

		a.loadPreviewImage(inputPath)
	}, a.window)
	fdialog.Show()
}

func (a *App) loadPreviewImage(inputPath string) {
	if a.currentPreset == nil {
		a.currentPreset = &models.Preset{
			Width:      800,
			Height:     600,
			Format:     "webp",
			Quality:    85,
			Mode:       "fit",
			Saturation: a.saturationSlider.Value,
			Brightness: a.brightnessSlider.Value,
			Contrast:   a.contrastSlider.Value,
		}
	}

	go func() {
		preview, err := a.imageProcessor.PreviewImage(inputPath, a.currentPreset, 500)
		if err != nil {
			fyne.Do(func() {
				a.statusLabel.Text = "Preview error: " + err.Error()
			})
			return
		}

		fyne.Do(func() {
			a.previewMutex.Lock()
			a.previewImage.Image = preview
			a.previewMutex.Unlock()
			a.previewImage.Refresh()
			a.statusLabel.Text = "Preview loaded: " + filepath.Base(inputPath)
		})
	}()
}

func hashURL(url string) string {
	hash := sha256.Sum256([]byte(url))
	return hex.EncodeToString(hash[:8])
}

func SavePNG(img image.Image, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return png.Encode(file, img)
}
