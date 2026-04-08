package models

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	OutputPath    string   `json:"outputPath"`
	PresetIDs     []string `json:"presetIds"`
	ActivePreset  string   `json:"activePreset"`
	CreatedAt     int64    `json:"createdAt"`
}

type Preset struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	Format     string  `json:"format"`
	Quality    int     `json:"quality"`
	Mode       string  `json:"mode"`
	Saturation float64 `json:"saturation"`
	Brightness float64 `json:"brightness"`
	Contrast   float64 `json:"contrast"`
}

type QueueItem struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Status    string `json:"status"`
	Progress  int    `json:"progress"`
	Error     string `json:"error"`
	FileName  string `json:"fileName"`
	ProjectID string `json:"projectId"`
	PresetID  string `json:"presetId"`
}

type QueueItemStatus struct {
	Pending     string
	Downloading string
	Processing  string
	Done       string
	Error      string
}

func NewQueueItemStatus() QueueItemStatus {
	return QueueItemStatus{
		Pending:     "pending",
		Downloading: "downloading",
		Processing:  "processing",
		Done:       "done",
		Error:      "error",
	}
}

func NewProject(name, outputPath string) *Project {
	return &Project{
		ID:        uuid.New().String(),
		Name:      name,
		OutputPath: outputPath,
		PresetIDs: []string{},
		CreatedAt: time.Now().Unix(),
	}
}

func NewPreset(name string, width, height, quality int, format, mode string) *Preset {
	return &Preset{
		ID:         uuid.New().String(),
		Name:       name,
		Width:      width,
		Height:     height,
		Format:     format,
		Quality:    quality,
		Mode:       mode,
		Saturation: 0,
		Brightness: 0,
		Contrast:   0,
	}
}

func NewQueueItem(url, projectID, presetID string) *QueueItem {
	return &QueueItem{
		ID:        uuid.New().String(),
		URL:       url,
		Status:    NewQueueItemStatus().Pending,
		ProjectID: projectID,
		PresetID:  presetID,
	}
}
