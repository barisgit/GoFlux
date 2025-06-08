package testassets

import (
	"embed"
	"testing"
)

// Safety test to ensure other tests don't confuse the tester if the test assets folder is removed
func TestAssets(t *testing.T) {
	assets := TestFS
	emptyFS := embed.FS{}

	// If empty throw error
	if assets == emptyFS {
		t.Error("TestFS is empty, check if you have accidentally removed the test assets folder, located in the 'goflux/internal/testassets' folder")
	}

	requiredFiles := []string{
		"assets/data.json",
		"assets/index.html",
		"assets/app.js",
		"assets/styles.css",
		"assets/images/icon.svg",
	}

	// If not empty, check if it contains the expected files
	for _, file := range requiredFiles {
		_, err := assets.ReadFile(file)
		if err != nil {
			t.Errorf("TestFS does not contain the expected file: %s, check if you have accidentally removed the test assets folder, located in the 'goflux/internal/testassets' folder", file)
		}
	}

	t.Log("TestFS contains the expected files")
}
