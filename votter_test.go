package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func Test_ListImages(t *testing.T) {
	tmpDir := t.TempDir()

	imageFiles := []string{"image1.jpg", "image2.png", "image3.jpg"}
	nonImageFiles := []string{"file1.txt", "file2.pdf"}

	for _, fileName := range append(imageFiles, nonImageFiles...) {
		file, err := os.Create(filepath.Join(tmpDir, fileName))
		if err != nil {
			t.Fatal(err)
		}
		file.Close()
	}

	images, err := listImages(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(images) != len(imageFiles) {
		t.Errorf("Expected %d images, found %d", len(imageFiles), len(images))
	}

	for _, img := range images {
		if !isImage(img) {
			t.Errorf("Expected %s to be an image file", img)
		}
	}
}

func Test_FindImages(t *testing.T) {
	rootDir := t.TempDir()
	labelDirs := []string{"label1", "label2"}
	imageFiles := []string{"image1.jpg", "image2.png"}

	for _, label := range labelDirs {
		labelDir := filepath.Join(rootDir, label)
		if err := os.Mkdir(labelDir, 0644); err != nil {
			t.Fatal(err)
		}
		for _, img := range imageFiles {
			file, err := os.Create(filepath.Join(labelDir, img))
			if err != nil {
				t.Fatal(err)
			}
			file.Close()
		}
	}

	labels, err := findImages(rootDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(labels) != len(labelDirs) {
		t.Errorf("Expected %d labels, found %d", len(labelDirs), len(labels))
	}

	for _, imgs := range labels {
		if len(imgs) != len(imageFiles) {
			t.Errorf("Expected %d images, found %d", len(imageFiles), len(imgs))
		}
	}
}

func Test_GenerateVottEntries(t *testing.T) {
	rootDir := t.TempDir()
	label := "label1"
	imageFile := "image1.jpg"
	labelDir := filepath.Join(rootDir, label)

	if err := os.Mkdir(labelDir, 0755); err != nil {
		t.Fatal(err)
	}
	imgPath := filepath.Join(labelDir, imageFile)
	file, err := os.Create(imgPath)
	if err != nil {
		t.Fatal(err)
	}
	file.Close()

	labels := map[string][]string{label: {imageFile}}

	entries, err := generateVottEntries(rootDir, labels)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 1 {
		t.Errorf("Expected 1 entry, found %d", len(entries))
	}

	entry := entries[0]
	if entry.Name != imageFile || entry.Label != label {
		t.Errorf("Expected entry with name %s and label %s, found %s and %s", imageFile, label, entry.Name, entry.Label)
	}
}

func Test_WriteVottJSON(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "vott-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer tmpFile.Close()

	assets := []Asset{
		{
			Format: "jpg",
			ID:     "id1",
			Name:   "image1.jpg",
			Path:   "file:/path/to/image1.jpg",
			Size:   Size{Width: 100, Height: 200},
			State:  0,
			Type:   0,
			Label:  "class_name",
		},
	}
	tags := []string{"class_name"}

	err = writeVottJSON(tmpFile.Name(), assets, tags)
	if err != nil {
		t.Fatal(err)
	}

	data, err := ioutil.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	var model VottJsonModel
	err = json.Unmarshal(data, &model)
	if err != nil {
		t.Fatal(err)
	}

	if len(model.Assets) != len(assets) {
		t.Errorf("Expected %d assets, found %d", len(assets), len(model.Assets))
	}
	if len(model.Tags) != len(tags) {
		t.Errorf("Expected %d tags, found %d", len(tags), len(model.Tags))
	}
}
