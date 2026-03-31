package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	aiURL     = "https://api.groq.com/openai/v1/chat/completions"
	githubURL = "https://api.github.com"

	aiModel      = "llama-3.3-70b-versatile"
	maxDiffLines = 500
	maxTokens    = 1024
	httpTimeout  = 30 * time.Second
	maxRetries   = 3
)

var prompt = `You are a senior software engineer performing a code review.
Review the following git diff and provide concise, actionable feedback.
Focus on: bugs, security issues, performance problems, and code clarity.
Format your response in markdown.`

var httpClient = &http.Client{Timeout: httpTimeout}

type aiRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type aiResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

type githubCommentRequest struct {
	Body string `json:"body"`
}

func main() {
	apiKey, err := mustEnv("GROQ_API_KEY")
	if err != nil {
		log.Fatalf("Invalid API key: %v", err)
	}
	githubToken, err := mustEnv("GITHUB_TOKEN")
	if err != nil {
		log.Fatalf("Failed to fetch github token: %v", err)
	}
	prNumber, err := mustEnv("PR_NUMBER")
	if err != nil {
		log.Fatalf("Failed to fetch PR number: %v", err)
	}
	repo, err := mustEnv("REPO")
	if err != nil {
		log.Fatalf("Failed to fetch repo name: %v", err)
	}

	diff, err := os.ReadFile("/tmp/pr_diff.txt")
	if err != nil {
		log.Fatalf("failed to read diff: %v", err)
	}
	if len(bytes.TrimSpace(diff)) == 0 {
		log.Println("diff is empty, nothing to review")
		return
	}

	review, err := callAI(apiKey, truncateDiff(string(diff)))
	if err != nil {
		log.Fatalf("AI review failed: %v", err)
	}

	if err := postGitHubComment(githubToken, repo, prNumber, review); err != nil {
		log.Fatalf("failed to post github comment: %v", err)
	}

	log.Println("review posted successfully")
}

func truncateDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	if len(lines) <= maxDiffLines {
		return diff
	}
	log.Printf("diff truncated from %d to %d lines to stay within API limits", len(lines), maxDiffLines)
	return strings.Join(lines[:maxDiffLines], "\n") + "\n\n[diff truncated — showing first 500 lines only]"
}

func callAI(apiKey, diff string) (string, error) {
	fullPrompt := fmt.Sprintf(`%s Diff: %s`, prompt, diff)
	request := aiRequest{
		Model:     aiModel,
		MaxTokens: maxTokens,
		Messages: []Message{
			{Role: "system", Content: "You are a senior software engineer performing code reviews."},
			{Role: "user", Content: fullPrompt},
		},
	}

	retryDelays := []time.Duration{10 * time.Second, 30 * time.Second, 60 * time.Second}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, rateLimited, err := requestAIReview(apiKey, request)
		if err != nil {
			return "", err
		}

		if rateLimited && attempt < maxRetries {
			log.Printf("rate limited (429) — retrying in %s (attempt %d/%d)",
				retryDelays[attempt], attempt+1, maxRetries)
			time.Sleep(retryDelays[attempt])
			continue
		}

		if result == nil || len(result.Choices) == 0 {
			return "", fmt.Errorf("AI returned no content")
		}

		return result.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("AI review failed after %d attempts", maxRetries)
}

func requestAIReview(apiKey string, request aiRequest) (*aiResponse, bool, error) {
	body, err := json.Marshal(request)
	if err != nil {
		return nil, false, err
	}
	req, err := http.NewRequest(http.MethodPost, aiURL, bytes.NewReader(body))
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, true, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("failed to get AI review, getting response code %d", resp.StatusCode)
	}

	var result aiResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, false, err
	}

	return &result, false, nil
}
func postGitHubComment(token, repo, prNumber, body string) error {
	url := fmt.Sprintf("%s/repos/%s/issues/%s/comments", githubURL, repo, prNumber)

	payload, err := json.Marshal(githubCommentRequest{Body: "## AI Code Review\n\n" + body})
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

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("github API returned status %d", resp.StatusCode)
	}
	return nil
}

func mustEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return v, nil
}
