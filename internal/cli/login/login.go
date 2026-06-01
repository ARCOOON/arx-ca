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

// Options configures the login flow. Non-empty Username and Password skip prompts.
// When ServerURL is set via flag, the server URL prompt is skipped unless stdin is a TTY.
type Options struct {
	ServerURL string
	Username  string
	Password  string
}

// Run prompts for admin credentials (unless provided in opts), logs in, and saves the JWT locally.
func Run(opts Options) error {
	url := strings.TrimSpace(opts.ServerURL)
	if url == "" {
		return fmt.Errorf("server URL is required; set server_url in ~/.arx/cli.yaml or pass --url")
	}

	reader := bufio.NewReader(os.Stdin)
	username := strings.TrimSpace(opts.Username)
	password := opts.Password

	if username == "" || password == "" {
		if !term.IsTerminal(int(syscall.Stdin)) && username != "" && password == "" {
			return fmt.Errorf("password is required when stdin is not a terminal (use --password)")
		}
	}

	if username == "" || (password == "" && term.IsTerminal(int(syscall.Stdin))) {
		if !skipServerURLPrompt(opts) {
			fmt.Printf("Server URL [%s]: ", url)
			line, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("read server url: %w", err)
			}
			if trimmed := strings.TrimSpace(line); trimmed != "" {
				url = trimmed
			}
		}

		if username == "" {
			fmt.Print("Username: ")
			line, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("read username: %w", err)
			}
			username = strings.TrimSpace(line)
			if username == "" {
				return fmt.Errorf("username is required")
			}
		}
	}

	if password == "" {
		var err error
		password, err = readPassword(reader)
		if err != nil {
			return err
		}
	}
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

func skipServerURLPrompt(opts Options) bool {
	return strings.TrimSpace(opts.ServerURL) != "" &&
		strings.TrimSpace(opts.Username) != "" &&
		opts.Password != ""
}

func readPassword(reader *bufio.Reader) (string, error) {
	fmt.Print("Password: ")
	if term.IsTerminal(int(syscall.Stdin)) {
		passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("read password: %w", err)
		}
		return string(passwordBytes), nil
	}

	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	return strings.TrimSpace(line), nil
}
