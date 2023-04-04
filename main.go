package main

import (
	"log"

	"github.com/go-git/go-git/v5"
)

func main() {
	if err := mainImpl(); err != nil {
		log.Fatal(err)
	}
}

// main creates a commit on a test branch in the current repository and pushes
// it. If the push is to an https://github.com repository, it uses
// $GITHUB_TOKEN and my username as auth info.
func mainImpl() error {
	r, err := git.PlainOpen(".git")
	if err != nil {
		return err
	}
	log.Print(r)

	return nil
}
