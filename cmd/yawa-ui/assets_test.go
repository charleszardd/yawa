package main

import (
	"io/fs"
	"net/http/httptest"
	"strings"
	"testing"

	wailsassets "github.com/wailsapp/wails/v2/pkg/assetserver"
	assetoptions "github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

func TestEmbeddedFrontendServesIndex(t *testing.T) {
	frontendAssets, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		t.Fatalf("fs.Sub() error = %v", err)
	}
	handler, err := wailsassets.NewAssetHandler(assetoptions.Options{Assets: frontendAssets}, nil)
	if err != nil {
		t.Fatalf("NewAssetHandler() error = %v", err)
	}

	request := httptest.NewRequest("GET", "http://wails.localhost/", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 200 {
		t.Fatalf("GET / status = %d, body = %q", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "Loading yawa") {
		t.Fatalf("GET / did not return embedded index: %q", response.Body.String())
	}
}
