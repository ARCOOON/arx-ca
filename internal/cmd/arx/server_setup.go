package arxcmd

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/spf13/cobra"

	"github.com/ARCOOON/arx-ca/internal/server/service"
)

const (
	defaultSetupRunAsUser  = "arx-ca"
	defaultSetupInstallDir = "/opt/arx"
)

func newServerSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Interactive wizard to install the Arx CA server as a systemd service",
		Long: `Guided installation of the arx binary, server.yaml, and arx-server systemd unit.
Requires root on Linux. Declining the systemd prompt exits without changes.`,
		Run: func(_ *cobra.Command, _ []string) {
			requireRootForService("server setup")
			opts, err := runInteractiveSetup(os.Stdin, os.Stdout, os.Stderr)
			if err != nil {
				log.Fatal(err)
			}
			if opts == nil {
				return
			}
			runServerServiceInstall(*opts)
		},
	}
}

func runInteractiveSetup(in io.Reader, out, errOut io.Writer) (*service.InstallOptions, error) {
	reader := bufio.NewReader(in)

	install, err := promptYesNo(reader, out, errOut, "Install ARX Certificate Authority as a systemd service? [Y/n]: ", true)
	if err != nil {
		return nil, fmt.Errorf("read confirmation: %w", err)
	}
	if !install {
		fmt.Fprintln(out, "Setup cancelled. No changes were made.")
		return nil, nil
	}

	runAsUser, err := promptWithDefault(reader, out, errOut, "Service User", defaultSetupRunAsUser, validateRunAsUser)
	if err != nil {
		return nil, err
	}

	installDir, err := promptWithDefault(reader, out, errOut, "Install Directory", defaultSetupInstallDir, validateInstallDir)
	if err != nil {
		return nil, err
	}

	opts := service.InstallOptions{
		RunAsUser:  runAsUser,
		InstallDir: installDir,
	}
	return &opts, nil
}

func promptYesNo(reader *bufio.Reader, out, errOut io.Writer, prompt string, defaultYes bool) (bool, error) {
	for {
		fmt.Fprint(out, prompt)
		line, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		answer := strings.TrimSpace(strings.ToLower(line))
		switch answer {
		case "", "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			if defaultYes {
				fmt.Fprintln(errOut, "Invalid input. Enter Y or n (default: Y).")
			} else {
				fmt.Fprintln(errOut, "Invalid input. Enter y or N (default: N).")
			}
		}
	}
}

func promptWithDefault(reader *bufio.Reader, out, errOut io.Writer, label, defaultValue string, validate func(string) error) (string, error) {
	prompt := fmt.Sprintf("%s [default: %s]: ", label, defaultValue)
	for {
		fmt.Fprint(out, prompt)
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("read %s: %w", strings.ToLower(label), err)
		}
		value := strings.TrimSpace(line)
		if value == "" {
			value = defaultValue
		}
		if err := validate(value); err != nil {
			fmt.Fprintf(errOut, "Invalid %s: %v\n", strings.ToLower(label), err)
			continue
		}
		return value, nil
	}
}

func validateRunAsUser(user string) error {
	if user == "" {
		return fmt.Errorf("cannot be empty")
	}
	if len(user) > 32 {
		return fmt.Errorf("must be at most 32 characters")
	}
	if user[0] == '-' || user[0] == '.' {
		return fmt.Errorf("must not start with '-' or '.'")
	}
	for _, r := range user {
		if r == '/' || r == ':' || unicode.IsSpace(r) {
			return fmt.Errorf("must not contain '/', ':', or whitespace")
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-' {
			return fmt.Errorf("may only contain letters, digits, '_', and '-'")
		}
	}
	return nil
}

func validateInstallDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("cannot be empty")
	}
	cleaned := filepath.Clean(dir)
	if !filepath.IsAbs(cleaned) {
		return fmt.Errorf("must be an absolute path")
	}
	if cleaned == "/" {
		return fmt.Errorf("cannot be the filesystem root")
	}
	return nil
}
