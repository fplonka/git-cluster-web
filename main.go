package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func runGitCommand(repoPath string, command ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", repoPath}, command...)...)
	return cmd.CombinedOutput()
}

func cloneRepo(repoURL, tempDir string) error {
	fmt.Println("cloning to", tempDir)
	cmd := exec.Command("git", "clone", repoURL, tempDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to clone repo: %v, output: %s", err, output)
	}
	fmt.Println("cloning done")
	return nil
}

func parseGitOutput(output []byte) map[string][]string {
	lines := strings.Split(string(output), "\n")
	commits := make(map[string][]string)
	var currentCommit string

	for _, line := range lines {
		if strings.HasPrefix(line, "__commit__:") {
			currentCommit = strings.Split(line, "__commit__:")[1]
		} else if line != "" {
			commits[line] = append(commits[line], currentCommit)
		}
	}
	return commits
}

func handler(w http.ResponseWriter, r *http.Request) {
	repoURL := r.URL.Query().Get("url")
	if repoURL == "" {
		http.Error(w, "url parameter is required", http.StatusBadRequest)
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

	output, err := runGitCommand(tempDir, "log", "--pretty=format:__commit__:%H", "--name-only")
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to run git command: %v", err), http.StatusInternalServerError)
		return
	}

	commits := parseGitOutput(output)
	jsonOutput, err := json.Marshal(commits)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to marshal json: %v", err), http.StatusInternalServerError)
		return
	}

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

func main() {
	http.HandleFunc("/clone", handler)
	// Serve files without caching
	http.Handle("/", noCacheHandler(http.FileServer(http.Dir("./static"))))
	fmt.Println("Server is listening on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
