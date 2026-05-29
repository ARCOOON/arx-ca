package login

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"github.com/your-org/arx-ca/internal/cli/api"
	"github.com/your-org/arx-ca/internal/cli/config"
	"github.com/your-org/arx-ca/internal/models"
)

const defaultServerURL = "http://localhost:8080"

// Run prompts for admin credentials, logs in, and saves the JWT locally.
func Run(serverURL string) error {
	url := strings.TrimSpace(serverURL)
	if url == "" {
		url = strings.TrimSpace(os.Getenv("ARX_SERVER_URL"))
	}
	if url == "" {
		url = defaultServerURL
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Server URL [%s]: ", url)
	line, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read server url: %w", err)
	}
	if trimmed := strings.TrimSpace(line); trimmed != "" {
		url = trimmed
	}

	fmt.Print("Username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("read username: %w", err)
	}
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("username is required")
	}

	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("read password: %w", err)
	}
	password := string(passwordBytes)
	if password == "" {
		return fmt.Errorf("password is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := api.Login(ctx, url, models.LoginRequest{
		Username: username,
		Password: password,
	})
	if err != nil {
		return err
	}

	tokenType := resp.TokenType
	if tokenType == "" {
		tokenType = "Bearer"
	}

	cfg := &config.Config{
		ServerURL: url,
		Token:     resp.Token,
		TokenType: tokenType,
		ExpiresAt: resp.ExpiresAt,
		Username:  username,
	}
	if err := config.Save(cfg); err != nil {
		return err
	}

	path, _ := config.Path()
	fmt.Printf("Logged in as %s. Token saved to %s\n", username, path)
	if !resp.ExpiresAt.IsZero() {
		fmt.Printf("Expires: %s\n", resp.ExpiresAt.Format(time.RFC3339))
	}
	if len(resp.Roles) > 0 {
		fmt.Printf("Roles: %s\n", strings.Join(resp.Roles, ", "))
	}
	return nil
}
