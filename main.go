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
	// Créer un logger pour écrire les messages d'erreur
	logFile, err := os.OpenFile("action.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Erreur lors de la création du fichier de log : %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	logger := log.New(multiWriter, "", log.LstdFlags)

	// Récupérer les variables d'environnement
	issueNumberStr := os.Getenv("ISSUE_NUMBER")
	issueTitle := os.Getenv("ISSUE_TITLE")
	labelsStr := os.Getenv("ISSUE_LABELS")

	giteaToken := os.Getenv("GITEA_TOKEN")
	giteaAPIURL := os.Getenv("GITEA_URL")
	repoOwner := os.Getenv("REPO_OWNER")
	repoName := os.Getenv("REPO_NAME")

	if issueNumberStr == "" || issueTitle == "" || giteaToken == "" || giteaAPIURL == "" || repoOwner == "" || repoName == "" {
		logger.Println("Erreur : certaines variables d'environnement sont manquantes.")
		os.Exit(1)
	}

	// Convertir le numéro de l'issue en int64
	issueNumber, err := strconv.ParseInt(issueNumberStr, 10, 64)
	if err != nil {
		logger.Printf("Erreur lors de la conversion du numéro de l'issue : %v\n", err)
		os.Exit(1)
	}

	// Mapper les labels aux préfixes Git Flow
	prefix := "feature"
	if labelsStr != "" {
		prefix, err = getPrefixFromLabels(labelsStr)
	}
	if err != nil {
		logger.Printf("Erreur lors de la détermination du préfixe : %v\n", err)
		os.Exit(1)
	}

	// Former le nom de la branche au format "prefixe/us-numero"
	branchName := fmt.Sprintf("%s/us-%d", prefix, issueNumber)

	logger.Printf("Nom de la branche défini : %s\n", branchName)

	// Configurer Git
	logger.Println("Configuration de Git...")
	err = runCommand(logger, "git", "config", "--global", "user.name", "gitea-actions")
	if err != nil {
		logger.Printf("Erreur lors de la configuration de Git : %v\n", err)
		os.Exit(1)
	}

	err = runCommand(logger, "git", "fetch", "origin")
	if err != nil {
		logger.Printf("Erreur lors de la mise a jour des references distantes")
		os.Exit(1)
	}

	err = runCommand(logger, "git", "config", "--global", "user.email", "actions@gitea.com")
	if err != nil {
		logger.Printf("Erreur lors de la configuration de Git : %v\n", err)
		os.Exit(1)
	}

	// Vérifier si la branche existe déjà
	logger.Printf("Vérification de l'existence de la branche '%s'...\n", branchName)
	output, err := exec.Command("git", "ls-remote", "--heads", "origin", branchName).CombinedOutput()
	if err != nil {
		logger.Printf("Erreur lors de la vérification de la branche : %v\nSortie : %s\n", err, string(output))
		os.Exit(1)
	}

	branchExists := false
	if strings.TrimSpace(string(output)) != "" {
		branchExists = true
	}

	if branchExists {
		logger.Printf("La branche '%s' existe déjà. Aucune action n'a été effectuée.\n", branchName)
	} else {
		// Créer et pousser la branche
		logger.Printf("Création de la branche '%s' à partir de 'dev'.\n", branchName)
		err = runCommand(logger, "git", "checkout", "-b", branchName, "origin/develop")
		if err != nil {
			logger.Printf("Erreur lors de la création de la branche : %v\n", err)
			os.Exit(1)
		}

		err = runCommand(logger, "git", "push", "origin", branchName)
		if err != nil {
			logger.Printf("Erreur lors du push de la branche : %v\n", err)
			os.Exit(1)
		}

		logger.Printf("Branche '%s' créée et poussée avec succès.\n", branchName)
	}

	// Attribuer la branche à l'issue via l'API Gitea
	logger.Printf("Attribution de la branche '%s' à l'issue #%d.\n", branchName, issueNumber)

	// Création du client Gitea
	client, err := gitea.NewClient(giteaAPIURL, gitea.SetToken(giteaToken))
	if err != nil {
		logger.Printf("Erreur lors de la création du client Gitea : %v\n", err)
		os.Exit(1)
	}

	// Préparer les options pour éditer l'issue
	editIssueOption := gitea.EditIssueOption{
		Ref: &branchName,
	}

	// Mettre à jour l'issue avec la référence de la branche
	_, _, err = client.EditIssue(repoOwner, repoName, issueNumber, editIssueOption)
	if err != nil {
		logger.Printf("Erreur lors de l'attribution de la branche à l'issue : %v\n", err)
		os.Exit(1)
	}

	logger.Printf("Branche '%s' attribuée à l'issue #%d avec succès.\n", branchName, issueNumber)
}

// Fonction pour mapper les labels aux préfixes
func getPrefixFromLabels(labelsStr string) (string, error) {
	// Définir la correspondance label ➔ préfixe
	labelPrefixMap := map[string]string{
		"enhancement": "feature",
		"invalid":     "bugfix",
		"bug":         "hotfix",
	}

	// Séparer les labels (supposant qu'ils sont séparés par des virgules)
	labels := strings.Split(labelsStr, ",")

	// Créer un mapping avec des clés en minuscules
	labelPrefixMapLower := make(map[string]string)
	for key, value := range labelPrefixMap {
		keyLower := strings.ToLower(key)
		labelPrefixMapLower[keyLower] = value
	}

	for _, label := range labels {
		label = strings.TrimSpace(label)
		labelLower := strings.ToLower(label)
		fmt.Printf("Label actuel : %s", labelLower)
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
	logger.Printf("Exécution de la commande : %s %s\n", name, strings.Join(arg, " "))
	return cmd.Run()
}
