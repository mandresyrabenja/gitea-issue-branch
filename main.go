package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"code.gitea.io/sdk/gitea"
)

func main() {
	// Create a logger to write error messages
	logFile, err := os.OpenFile("action.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Error while creating the log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger := log.New(multiWriter, "", log.LstdFlags)

	// Retrieve environment variables
	issueNumberStr := os.Getenv("ISSUE_NUMBER")
	issueTitle := os.Getenv("ISSUE_TITLE")
	labelsStr := os.Getenv("ISSUE_LABELS")

	giteaToken := os.Getenv("GITEA_TOKEN")
	giteaAPIURL := os.Getenv("GITEA_URL")
	repoOwner := os.Getenv("REPO_OWNER")
	repoName := os.Getenv("REPO_NAME")

	if issueNumberStr == "" || issueTitle == "" || giteaToken == "" || giteaAPIURL == "" || repoOwner == "" || repoName == "" {
		logger.Println("Error: some environment variables are missing.")
		os.Exit(1)
	}

	// Convert the issue number to int64
	issueNumber, err := strconv.ParseInt(issueNumberStr, 10, 64)
	if err != nil {
		logger.Printf("Error while converting the issue number: %v\n", err)
		os.Exit(1)
	}

	// Map labels to Git Flow prefixes
	prefix := "feature"
	if labelsStr != "" {
		prefix, err = getPrefixFromLabels(labelsStr)
	}
	if err != nil {
		logger.Printf("Error while determining the prefix: %v\n", err)
		os.Exit(1)
	}

	// Form the branch name in the format "prefix/ticket-number"
	branchName := fmt.Sprintf("%s/ticket-%d", prefix, issueNumber)

	logger.Printf("Branch name set to: %s\n", branchName)

	// Configure Git
	logger.Println("Configuring Git...")
	err = runCommand(logger, "git", "config", "--global", "user.name", "gitea-actions")
	if err != nil {
		logger.Printf("Error while configuring Git: %v\n", err)
		os.Exit(1)
	}

	err = runCommand(logger, "git", "fetch", "origin")
	if err != nil {
		logger.Printf("Error while updating remote references")
		os.Exit(1)
	}

	err = runCommand(logger, "git", "config", "--global", "user.email", "actions@gitea.com")
	if err != nil {
		logger.Printf("Error while configuring Git: %v\n", err)
		os.Exit(1)
	}

	// Check if the branch already exists
	logger.Printf("Checking if branch '%s' exists...\n", branchName)
	output, err := exec.Command("git", "ls-remote", "--heads", "origin", branchName).CombinedOutput()
	if err != nil {
		logger.Printf("Error while checking the branch: %v\nOutput: %s\n", err, string(output))
		os.Exit(1)
	}

	branchExists := false
	if strings.TrimSpace(string(output)) != "" {
		branchExists = true
	}

	if branchExists {
		logger.Printf("Branch '%s' already exists. No action taken.\n", branchName)
	} else {
		// Create and push the branch
		logger.Printf("Creating branch '%s' from 'develop'.\n", branchName)
		err = runCommand(logger, "git", "checkout", "-b", branchName, "origin/develop")
		if err != nil {
			logger.Printf("Error while creating the branch: %v\n", err)
			os.Exit(1)
		}

		err = runCommand(logger, "git", "push", "origin", branchName)
		if err != nil {
			logger.Printf("Error while pushing the branch: %v\n", err)
			os.Exit(1)
		}

		logger.Printf("Branch '%s' successfully created and pushed.\n", branchName)
	}

	// Assign the branch to the issue via the Gitea API
	logger.Printf("Assigning branch '%s' to issue #%d.\n", branchName, issueNumber)

	// Create the Gitea client
	client, err := gitea.NewClient(giteaAPIURL, gitea.SetToken(giteaToken))
	if err != nil {
		logger.Printf("Error while creating the Gitea client: %v\n", err)
		os.Exit(1)
	}

	// Prepare options to edit the issue
	editIssueOption := gitea.EditIssueOption{
		Ref: &branchName,
	}

	// Update the issue with the branch reference
	_, _, err = client.EditIssue(repoOwner, repoName, issueNumber, editIssueOption)
	if err != nil {
		logger.Printf("Error while assigning the branch to the issue: %v\n", err)
		os.Exit(1)
	}

	logger.Printf("Branch '%s' successfully assigned to issue #%d.\n", branchName, issueNumber)
}

// Function to map labels to prefixes
func getPrefixFromLabels(labelsStr string) (string, error) {
	// Define the label ➔ prefix correspondence
	labelPrefixMap := map[string]string{
		"enhancement": "feature",
		"invalid":     "bugfix",
		"bug":         "hotfix",
	}

	// Split labels (assuming they are comma-separated)
	labels := strings.Split(labelsStr, ",")

	// Create a mapping with lower-case keys
	labelPrefixMapLower := make(map[string]string)
	for key, value := range labelPrefixMap {
		keyLower := strings.ToLower(key)
		labelPrefixMapLower[keyLower] = value
	}

	for _, label := range labels {
		label = strings.TrimSpace(label)
		labelLower := strings.ToLower(label)
		fmt.Printf("Current label: %s", labelLower)
		if prefix, exists := labelPrefixMapLower[labelLower]; exists {
			return prefix, nil
		}
	}

	return "feature", nil
}

func runCommand(logger *log.Logger, name string, arg ...string) error {
	cmd := exec.Command(name, arg...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	logger.Printf("Executing command: %s %s\n", name, strings.Join(arg, " "))
	return cmd.Run()
}
