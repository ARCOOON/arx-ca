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
	"github.com/your-org/arx-ca/internal/cli/runtime"
	arxconfig "github.com/your-org/arx-ca/internal/config"
	"github.com/your-org/arx-ca/internal/models"
)

// Options configures the login flow. Non-empty Email and Password skip prompts.
// When ServerURL is set via flag, the server URL prompt is skipped unless stdin is a TTY.
type Options struct {
	ServerURL string
	Email     string
	Password  string
}

// Run prompts for admin credentials (unless provided in opts), logs in, and saves the JWT locally.
func Run(opts Options) error {
	url, err := runtime.ResolveServerURL(runtime.URLResolveOptions{
		FlagOverride: opts.ServerURL,
	})
	if err != nil {
		return err
	}

	reader := bufio.NewReader(os.Stdin)
	email := strings.TrimSpace(opts.Email)
	password := opts.Password

	if email == "" || password == "" {
		if !term.IsTerminal(int(syscall.Stdin)) && email != "" && password == "" {
			return fmt.Errorf("password is required when stdin is not a terminal (use --password)")
		}
	}

	if email == "" || (password == "" && term.IsTerminal(int(syscall.Stdin))) {
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

		if email == "" {
			fmt.Print("Email: ")
			line, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("read email: %w", err)
			}
			email = strings.TrimSpace(line)
			if email == "" {
				return fmt.Errorf("email is required")
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
		Email:    email,
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
		Email:     email,
	}
	if err := config.Save(cfg); err != nil {
		return err
	}
	if err := arxconfig.SetCLIServerURL(url); err != nil {
		return err
	}

	path, _ := config.Path()
	fmt.Printf("Logged in as %s. Token saved to %s\n", email, path)
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
		strings.TrimSpace(opts.Email) != "" &&
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
