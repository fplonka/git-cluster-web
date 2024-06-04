package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	CacheDir      = "./cache"
	CacheDuration = 7 * 24 * time.Hour // 1 week
)

func runGitCommand(print bool, command ...string) ([]byte, error) {
	cmd := exec.Command("git", command...)

	// tell the command to print its output to stdout
	if print {
		var stdBuffer bytes.Buffer

		cmd.Stdout = &stdBuffer
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			log.Panic(err)
		}

		return stdBuffer.Bytes(), nil
	} else {
		return cmd.CombinedOutput()
	}
}

// create the commit map given the dir we cloned the repo to
func createCommitMap(tempDir string) (map[string][]string, error) {
	cmd := exec.Command("git", "-C", tempDir, "log", "--pretty=format:__commit__:%H", "--name-only")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	commits := make(map[string][]string)
	scanner := bufio.NewScanner(stdout)
	var currentCommit string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "__commit__:") {
			currentCommit = strings.TrimPrefix(line, "__commit__:")
		} else if line != "" {
			commits[line] = append(commits[line], currentCommit)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading command output: %w", err)
	}

	if err := cmd.Wait(); err != nil {
		return nil, fmt.Errorf("command execution failed: %w", err)
	}

	return commits, nil
}

func getCurrentFiles(repoPath string) (map[string]struct{}, error) {
	output, err := runGitCommand(false, "-C", repoPath, "ls-tree", "-r", "HEAD", "--name-only")
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(output), "\n")
	currentFiles := make(map[string]struct{}, len(lines))
	for _, line := range lines {
		if line != "" {
			currentFiles[line] = struct{}{}
		}
	}
	return currentFiles, nil
}

func cloneRepo(repoURL, tempDir string) error {
	output, err := runGitCommand(true, "clone", "--no-checkout", repoURL, tempDir)
	if err != nil {
		fmt.Println("ERR:", err)
		return fmt.Errorf("failed to clone repo: %v, output: %s", err, output)
	}
	return nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	repoURL := r.URL.Query().Get("url")
	if repoURL == "" {
		http.Error(w, "url parameter is required", http.StatusBadRequest)
		return
	}

	repoHash := fmt.Sprintf("%x", md5.Sum([]byte(repoURL)))
	cachePath := filepath.Join(CacheDir, repoHash)

	// Check if the cache exists
	if _, err := os.Stat(cachePath); err == nil {
		fmt.Println("found in cache!")
		http.ServeFile(w, r, cachePath)
		return
	}

	tempDir, err := os.MkdirTemp("", "repo")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create temp dir: %v", err), http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tempDir)

	if err := cloneRepo(repoURL, tempDir); err != nil {
		http.Error(w, fmt.Sprintf("failed to clone repo: %v", err), http.StatusInternalServerError)
		return
	}
	fmt.Println("cloning done...")

	commits, err := createCommitMap(tempDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to parse repo data: %v", err), http.StatusInternalServerError)
		fmt.Printf("PARSING ERROR?!: %v\n", err)
	}
	fmt.Println("output parsed")

	currentFiles, err := getCurrentFiles(tempDir)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get current files: %v", err), http.StatusInternalServerError)
		return
	}
	for file, hashes := range commits {
		if _, exists := currentFiles[file]; !exists || len(hashes) <= 1 {
			delete(commits, file)
		}
	}

	jsonOutput, err := json.Marshal(commits)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to marshal json: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Println()

	// Save the result to cache
	os.WriteFile(cachePath, jsonOutput, 0644)

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonOutput)
}

func noCacheHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set headers to prevent caching
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate, max-age=0")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")
		w.Header().Set("Surrogate-Control", "no-store")

		// Serve the request
		h.ServeHTTP(w, r)
	})
}

func cleanupOldCache() {
	files, err := os.ReadDir(CacheDir)
	if err != nil {
		log.Printf("failed to read cache directory: %v", err)
		return
	}

	now := time.Now()
	for _, file := range files {
		info, err := file.Info()
		if err != nil {
			log.Printf("failed to get file info: %v", err)
			continue
		}

		if now.Sub(info.ModTime()) > CacheDuration {
			os.RemoveAll(filepath.Join(CacheDir, file.Name()))
		}
	}
}

func init() {
	os.MkdirAll(CacheDir, 0755)
	// cleanupOldCache()
}

func main() {
	http.HandleFunc("/clone", handler)
	// Serve files without caching
	http.Handle("/", noCacheHandler(http.FileServer(http.Dir("./static"))))
	fmt.Println("Server is listening on port 8080")
	log.Fatal(http.ListenAndServe(":8282", nil))
}
