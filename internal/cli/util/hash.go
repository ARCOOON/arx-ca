package util

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/bcrypt"
)

// NewHashCmd returns the bcrypt password hashing subcommand.
func NewHashCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hash <password>",
		Short: "Generate a bcrypt hash for a password",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			hash, err := bcrypt.GenerateFromPassword([]byte(args[0]), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("hash password: %w", err)
			}
			_, err = fmt.Fprintln(os.Stdout, string(hash))
			return err
		},
	}
}
