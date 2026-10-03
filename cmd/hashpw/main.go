// hashpw creates the bcrypt hash for ADMIN_PASSWORD_HASH.
//
//	go run ./cmd/hashpw
//
// The password is read from stdin (not from a command-line argument) so it
// does not end up in the shell history.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	fmt.Fprint(os.Stderr, "Admin-Passwort: ")
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && line == "" {
		fmt.Fprintln(os.Stderr, "no input")
		os.Exit(1)
	}
	password := strings.TrimRight(line, "\r\n")
	if len(password) < 12 {
		fmt.Fprintln(os.Stderr, "Passwort muss mindestens 12 Zeichen haben.")
		os.Exit(1)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(string(hash))
}
