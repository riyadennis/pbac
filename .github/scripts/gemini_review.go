package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

const (
	geminiURL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent"
	githubURL = "https://api.github.com"
)

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

type githubCommentRequest struct {
	Body string `json:"body"`
}

func main() {
	apiKey := mustEnv("GEMINI_API_KEY")
	githubToken := mustEnv("GITHUB_TOKEN")
	prNumber := mustEnv("PR_NUMBER")
	repo := mustEnv("REPO")

	diff, err := os.ReadFile("/tmp/pr_diff.txt")
	if err != nil {
		log.Fatalf("failed to read diff: %v", err)
	}
	if len(bytes.TrimSpace(diff)) == 0 {
		log.Println("diff is empty, nothing to review")
		return
	}

	review, err := callGemini(apiKey, string(diff))
	if err != nil {
		log.Fatalf("gemini review failed: %v", err)
	}

	if err := postGitHubComment(githubToken, repo, prNumber, review); err != nil {
		log.Fatalf("failed to post github comment: %v", err)
	}

	log.Println("review posted successfully")
}

func callGemini(apiKey, diff string) (string, error) {
	prompt := fmt.Sprintf(`You are a senior software engineer performing a code review.
Review the following git diff and provide concise, actionable feedback.
Focus on: bugs, security issues, performance problems, and code clarity.
Format your response in markdown.

Diff:
%s`, diff)

	body, err := json.Marshal(geminiRequest{
		Contents: []geminiContent{{
			Parts: []geminiPart{{Text: prompt}},
		}},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, geminiURL+"?key="+apiKey, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini API returned status %d", resp.StatusCode)
	}

	var result geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned no content")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}

func postGitHubComment(token, repo, prNumber, body string) error {
	url := fmt.Sprintf("%s/repos/%s/issues/%s/comments", githubURL, repo, prNumber)

	payload, err := json.Marshal(githubCommentRequest{Body: "## Gemini Code Review\n\n" + body})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("github API returned status %d", resp.StatusCode)
	}
	return nil
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return v
}