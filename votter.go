// Votter is a command-line tool for generating VoTT (Visual Object Tagging Tool) annotations in JSON format.
// Takes a folder of images labelled by directory name and writes a VoTT file with regions for the labels.
//
//	votter.exe [pathToImages] [vott-coco-annotations.json]
//	go run votter.go test/dataset test/dataset/vott-coca-annotations.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const Version = "1"
const OptionalPathToImagesDefault = "."
const OptionalAnnotationsFilenameDefault = "vott-coco-annotations.json"
const ExitSuccesful = 0
const ExitImagesFolderNotFound = 1
const ExitImagesFolderEmpty = 2
const ExitAnnotationsFolderNotFound = 3

type VottJsonModel struct {
	Name                   string                 `json:"name"`
	SecurityToken          string                 `json:"securityToken"`
	VideoSettings          VideoSettings          `json:"videoSettings"`
	Tags                   []Tag                  `json:"tags"`
	ID                     string                 `json:"id"`
	ActiveLearningSettings ActiveLearningSettings `json:"activeLearningSettings"`
	Version                string                 `json:"version"`
	LastVisitedAssetID     string                 `json:"lastVisitedAssetId"`
	Assets                 map[string]AssetDetail `json:"assets"`
}

type VideoSettings struct {
	FrameExtractionRate int `json:"frameExtractionRate"`
}

type Tag struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type ActiveLearningSettings struct {
	AutoDetect    bool   `json:"autoDetect"`
	PredictTag    bool   `json:"predictTag"`
	ModelPathType string `json:"modelPathType"`
}

type AssetDetail struct {
	Asset   Asset    `json:"asset"`
	Regions []Region `json:"regions"`
	Version string   `json:"version"`
}

type Asset struct {
	Format string `json:"format"`
	ID     string `json:"id"`
	Name   string `json:"name"`
	Path   string `json:"path"`
	Size   Size   `json:"size"`
	State  int    `json:"state"`
	Type   int    `json:"type"`
	Label  string
}

type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Region struct {
	ID          string      `json:"id"`
	Type        string      `json:"type"`
	Tags        []string    `json:"tags"`
	BoundingBox BoundingBox `json:"boundingBox"`
	Points      []Point     `json:"points"`
}

type BoundingBox struct {
	Height int `json:"height"`
	Width  int `json:"width"`
	Left   int `json:"left"`
	Top    int `json:"top"`
}

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func main() {

	// --- Step 1. Command line parameters ------------------------------------
	//
	// Command line flags for -v (version) and -h (help).
	versionFlag := flag.Bool("v", false, "Print version")
	helpFlag := flag.Bool("h", false, "Show help")
	flag.Parse()

	if *versionFlag {
		fmt.Println(Version)
		os.Exit(0)
	}

	if *helpFlag {
		flag.Usage()
		return
	}

	// Command line positional arguments for:  votter.exe <pathToImages> <vott-coco-annotations.json>
	args := flag.Args()
	imagesPath := OptionalPathToImagesDefault
	annotationFile := OptionalAnnotationsFilenameDefault

	if len(args) > 0 {
		imagesPath = args[0]
	}

	if len(args) == 2 {
		annotationFile = args[1]
	}

	// Verify the paths for images and annotations ara available.
	if !isDirectory(imagesPath) {
		fmt.Printf("Error: '%s' is not an existing directory\n", imagesPath)
		os.Exit(ExitImagesFolderNotFound)
	}

	if !isDirectory(filepath.Dir(annotationFile)) {
		fmt.Printf("Error: Cannot write annotations to directory '%s'\n", annotationFile)
		os.Exit(ExitAnnotationsFolderNotFound)
	}

	// --- Step 2. Generate VoTT assets --------------------------------------
	//
	// Find images in subdirectories, folder names are the labels.
	imagesPerLabelDirectoryMap, err := findImages(imagesPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(ExitImagesFolderEmpty)
	}

	// Make a distinct list of labels from the directory names found with the labeled images.
	var labels []string
	for label := range imagesPerLabelDirectoryMap {
		labels = append(labels, label)
	}

	// Generate VoTT assets with image names and regions.
	assets, err := generateVottEntries(imagesPath, imagesPerLabelDirectoryMap)
	if err != nil {
		fmt.Println(err)
		os.Exit(ExitImagesFolderEmpty)
	}

	// --- Step 3. Write JSON file --------------------------------------------
	//
	// Print label and image info to std out.
	for label, images := range imagesPerLabelDirectoryMap {
		for _, image := range images {
			fmt.Printf("Label '%s' for image '%s'.\n", label, image)
		}
	}

	// Write JSON file vott-cocoa-annotation.json
	if err := writeVottJSON(annotationFile, assets, labels); err != nil {
		fmt.Println(err)
		os.Exit(ExitImagesFolderNotFound)
	}

	os.Exit(ExitSuccesful)
}

// isDirectory checks if the given path is a directory.
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// findImages get all the labeled images in the given directory and its subdirectories. Returns a map of the directory name (label) to containing image paths.
func findImages(root string) (map[string][]string, error) {
	labels := make(map[string][]string)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != root {
			label := filepath.Base(path)
			images, err := listImages(path)
			if err != nil {
				return err
			}
			if len(images) > 0 {
				labels[label] = images
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if len(labels) == 0 {
		fmt.Print("Error: No images found in subdirectories.")
		os.Exit(ExitImagesFolderEmpty)
	}

	return labels, nil
}

func listImages(dir string) ([]string, error) {
	var images []string
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if isImage(file.Name()) {
			images = append(images, file.Name())
		}
	}
	return images, nil
}

func isImage(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" || ext == ".bmp"
}

func generateVottEntries(pathToImagesDataset string, labels map[string][]string) ([]Asset, error) {
	var entries []Asset

	for label, images := range labels {
		for _, imgFileName := range images {
			imgRelativePath := filepath.Join(pathToImagesDataset, label, imgFileName) // dataset/label/image.jpg
			imgAbsolutePath, err := filepath.Abs(imgRelativePath)                     // /home/example/dataset/label/image.jpg or C:\example\dataset\label\image.jpg
			if err != nil {
				return nil, err
			}

			imgFile, err := os.Open(imgRelativePath)
			if err != nil {
				return nil, err
			}
			imgConfig, _, err := image.DecodeConfig(imgFile)
			imgFile.Close()
			if err != nil {
				return nil, err
			}

			entry := Asset{
				Format: strings.TrimPrefix(filepath.Ext(imgFileName), "."),
				ID:     uuid.New().String(),
				Name:   imgFileName,
				Path:   "file:" + filepath.ToSlash(imgAbsolutePath), // file:/home/example/dataset/label/image.jpg or file:C:/example/dataset/label/image.jpg
				Size: Size{
					Width:  imgConfig.Width,
					Height: imgConfig.Height,
				},
				State: 0,
				Type:  0,
				Label: label,
			}
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

func writeVottJSON(path string, assets []Asset, tags []string) error {

	model := VottJsonModel{
		ActiveLearningSettings: ActiveLearningSettings{AutoDetect: false, PredictTag: true, ModelPathType: "coco"},
		Assets:                 make(map[string]AssetDetail),
		Tags:                   []Tag{},
		Version:                "2.2.0",
	}

	for _, asset := range assets {
		region := Region{
			ID:          uuid.New().String(),
			Type:        "RECTANGLE",
			Tags:        []string{asset.Label},
			BoundingBox: BoundingBox{Height: asset.Size.Height, Width: asset.Size.Width, Left: 0, Top: 0},
			Points:      []Point{{X: 0, Y: 0}, {X: asset.Size.Width, Y: asset.Size.Height}},
		}
		assetDetail := AssetDetail{
			Asset:   asset,
			Regions: []Region{region},
			Version: "2.2.0", // last version
		}
		model.Assets[asset.ID] = assetDetail
	}

	for _, label := range tags {
		tag := Tag{
			Name:  label,
			Color: "#ff0000", // red
		}
		model.Tags = append(model.Tags, tag)
	}

	data, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0644)
}
