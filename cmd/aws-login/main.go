package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/huh"
)

type AWSProfile struct {
	Name               string
	AWSAccountID       string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	Region             string
	RoleARN            string
	SourceProfile      string
}

const lastUsedFile = ".aws-profile-selector-last"

func main() {
	var useLastProfile bool
	var searchTerm string

	flag.BoolVar(&useLastProfile, "l", false, "Use the last saved profile")
	flag.StringVar(&searchTerm, "s", "", "Search term for profile selection")
	flag.Parse()

	profiles, err := loadProfiles()
	if err != nil {
		fmt.Printf("Error reading AWS credentials: %v\n", err)
		os.Exit(1)
	}

	var selectedProfile string

	if useLastProfile {
		selectedProfile = getLastUsedProfile()
		if selectedProfile == "" {
			fmt.Println("No last used profile found.")
			os.Exit(1)
		}
	} else if searchTerm != "" {
		selectedProfile = handleProfileSearch(profiles, searchTerm)
	}

	if selectedProfile == "" {
		var err error
		selectedProfile, err = showProfileSelectionPrompt(profiles)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	}

	if selectedProfile == "" {
		fmt.Println("No profile selected. Exiting.")
		os.Exit(1)
	}

	if err := selectAndUseProfile(selectedProfile); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func handleProfileSearch(profiles map[string]AWSProfile, searchTerm string) string {
	searchResults := searchProfiles(profiles, searchTerm)
	if len(searchResults) > 0 {
		suggestedProfile := searchResults[0]
		fmt.Printf("Use suggested profile \"%s\"? (y/n): ", suggestedProfile.Name)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) == "y" {
			return suggestedProfile.Name
		}
	} else {
		fmt.Println("No matching profiles found.")
	}
	return ""
}

func loadProfiles() (map[string]AWSProfile, error) {
	homeDir, _ := os.UserHomeDir()
	credentialsPath := filepath.Join(homeDir, ".aws", "credentials")
	content, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, err
	}
	return parseAWSCredentials(string(content)), nil
}

func getProfileEmoji(profileName string) string {
	if strings.Contains(profileName, "prod") {
		return "" // 🔴
	}
	if strings.Contains(profileName, "test") {
		return "" // 🟡
	}
	return "" // 🟢
}

func isValidProfileName(name string) bool {
	match, _ := regexp.MatchString("^[a-zA-Z0-9][a-zA-Z0-9_-]*$", name)
	return match
}

func parseAWSCredentials(content string) map[string]AWSProfile {
	profiles := make(map[string]AWSProfile)
	var currentProfile string

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			profileName := line[1 : len(line)-1]
			if isValidProfileName(profileName) && profileName != "default" {
				currentProfile = profileName
				profiles[currentProfile] = AWSProfile{Name: currentProfile}
			} else {
				currentProfile = ""
			}
		} else if currentProfile != "" && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			profile := profiles[currentProfile]
			switch key {
			case "aws_access_key_id":
				profile.AWSAccessKeyID = value
			case "aws_secret_access_key":
				profile.AWSSecretAccessKey = value
			case "aws_account_id":
				profile.AWSAccountID = value
			case "region":
				profile.Region = value
			case "role_arn":
				profile.RoleARN = value
			case "source_profile":
				profile.SourceProfile = value
			}
			profiles[currentProfile] = profile
		}
	}

	return profiles
}

func getLastUsedProfile() string {
	homeDir, _ := os.UserHomeDir()
	content, err := os.ReadFile(filepath.Join(homeDir, lastUsedFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

func saveLastUsedProfile(profileName string) error {
	homeDir, _ := os.UserHomeDir()
	return os.WriteFile(filepath.Join(homeDir, lastUsedFile), []byte(profileName), 0644)
}

func getCurrentRegion() string {
	cmd := exec.Command("aws", "configure", "get", "region")
	output, err := cmd.Output()
	if err != nil {
		return "Not set"
	}
	return strings.TrimSpace(string(output))
}

func searchProfiles(profiles map[string]AWSProfile, query string) []AWSProfile {
	query = strings.ToLower(query)
	var rankedProfiles []AWSProfile

	type profileScore struct {
		profile AWSProfile
		score   int
	}

	var scores []profileScore

	for name, profile := range profiles {
		score := rankProfile(name, query)
		if score > 0 {
			scores = append(scores, profileScore{profile: profile, score: score})
		}
	}

	// Sort profiles by score in descending order
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	for _, ps := range scores {
		rankedProfiles = append(rankedProfiles, ps.profile)
	}

	return rankedProfiles
}

func rankProfile(profileName, query string) int {
	profileName = strings.ToLower(profileName)
	terms := strings.Fields(query)
	score := 0

	for _, term := range terms {
		if strings.Contains(profileName, term) {
			score += 1
		}
	}

	return score
}

func showProfileSelectionPrompt(profiles map[string]AWSProfile) (string, error) {
	var options []huh.Option[string]

	var names []string
	for name := range profiles {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		profile := profiles[name]
		emoji := getProfileEmoji(name)
		displayName := fmt.Sprintf("%s %s (%s)", emoji, name, profile.AWSAccountID)
		options = append(options, huh.NewOption(displayName, name))
	}

	lastUsed := getLastUsedProfile()
	var selectedProfile string = lastUsed

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select an AWS profile").
				Options(options...).
				Value(&selectedProfile),
		),
	)

	err := form.Run()
	if err != nil {
		return "", err
	}

	return selectedProfile, nil
}

func selectAndUseProfile(profileName string) error {
	if err := saveLastUsedProfile(profileName); err != nil {
		return err
	}

	fmt.Printf("Selected profile: %s\n", profileName)

	newRegion := getCurrentRegion()
	fmt.Printf("New default region: %s\n", newRegion)

	useOnePassCLI := os.Getenv("USE_ONEPASS_CLI")
	var cmd *exec.Cmd

	if useOnePassCLI == "true" {
		cmd = exec.Command("op", "run", "--", "aws", "sts", "get-caller-identity")
	} else {
		cmd = exec.Command("aws", "sts", "get-caller-identity")
	}

	cmd.Env = append(os.Environ(), fmt.Sprintf("AWS_PROFILE=%s", profileName))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error executing AWS CLI command: %v", err)
	}

	fmt.Printf("Command output: %s\n", output)
	return nil
}
