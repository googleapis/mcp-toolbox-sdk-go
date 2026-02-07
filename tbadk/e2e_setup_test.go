//go:build e2e

// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tbadk_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"cloud.google.com/go/storage"
	"google.golang.org/api/idtoken"
)

// --- Helper Functions ---

func getEnvVar(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Fatal: Must set environment variable %s", key)
	}
	return value
}

func accessSecretVersion(ctx context.Context, projectID, secretID string, version string) string {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create secretmanager client: %v", err)
	}
	defer client.Close()

	req := &secretmanagerpb.AccessSecretVersionRequest{
		Name: fmt.Sprintf("projects/%s/secrets/%s/versions/%s", projectID, secretID, version),
	}

	result, err := client.AccessSecretVersion(ctx, req)
	if err != nil {
		log.Fatalf("Failed to access secret version '%s': %v", secretID, err)
	}
	return string(result.Payload.Data)
}

func downloadBlob(ctx context.Context, bucketName, sourceBlobName, destinationFileName string) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create storage client: %v", err)
	}
	defer client.Close()

	rc, err := client.Bucket(bucketName).Object(sourceBlobName).NewReader(ctx)
	if err != nil {
		log.Fatalf("Failed to create reader for blob %s: %v", sourceBlobName, err)
	}
	defer rc.Close()

	f, err := os.Create(destinationFileName)
	if err != nil {
		log.Fatalf("Failed to create destination file %s: %v", destinationFileName, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, rc); err != nil {
		log.Fatalf("Failed to download blob %s: %v", sourceBlobName, err)
	}
	log.Printf("Blob %s downloaded to %s.", sourceBlobName, destinationFileName)
}

func getToolboxBinaryURL(toolboxVersion string) string {
	osSystem := runtime.GOOS
	arch := runtime.GOARCH
	return fmt.Sprintf("v%s/%s/%s/toolbox", toolboxVersion, osSystem, arch)
}

func getAuthToken(ctx context.Context, clientID string) string {
	tokenSource, err := idtoken.NewTokenSource(ctx, clientID)
	if err != nil {
		log.Fatalf("Failed to create token source for audience %s: %v", clientID, err)
	}
	token, err := tokenSource.Token()
	if err != nil {
		log.Fatalf("Failed to retrieve token: %v", err)
	}
	return token.AccessToken
}

func setupAndStartToolboxServer(ctx context.Context, version, toolsFilePath string) *exec.Cmd {
	log.Println("Downloading toolbox binary from GCS bucket...")
	binaryURL := getToolboxBinaryURL(version)
	binaryPath := "toolbox"
	downloadBlob(ctx, "genai-toolbox", binaryURL, binaryPath)
	log.Println("Toolbox binary downloaded successfully.")

	if err := os.Chmod(binaryPath, 0755); err != nil {
		log.Fatalf("Failed to make toolbox binary executable: %v", err)
	}

	absBinaryPath, err := filepath.Abs(binaryPath)
	if err != nil {
		log.Fatalf("Failed to get absolute path for toolbox binary: %v", err)
	}

	log.Println("Starting toolbox server process...")
	cmd := exec.Command(absBinaryPath, "--tools-file", toolsFilePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start toolbox server: %v", err)
	}

	log.Println("Waiting for server to initialize...")
	// A more robust way to check for server readiness is to poll the health endpoint.
	// For now, Sleep is a simple way to wait.
	time.Sleep(5 * time.Second)

	if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
		log.Fatalf("Toolbox server failed to start and exited with code: %d", cmd.ProcessState.ExitCode())
	}

	log.Println("Toolbox server started successfully.")
	return cmd
}
