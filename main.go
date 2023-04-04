package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

func main() {
	opts := options{
		URL:    "https://github.com/spraints/silver-eureka",
		Branch: "testing-123",
		User:   "spraints",
		Token:  os.Getenv("GITHUB_TOKEN"),
	}

	if err := mainImpl(opts); err != nil {
		log.Fatal(err)
	}
}

type options struct {
	URL    string
	Branch string
	User   string
	Token  string
}

func mainImpl(opts options) error {
	tmpdir, err := os.MkdirTemp("", "silver-eureka-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)

	repoPath := filepath.Join(tmpdir, "testing")

	log.Printf("cloning %s to %q", opts.URL, repoPath)
	r, err := git.PlainClone(repoPath, false, &git.CloneOptions{URL: opts.URL})
	if err != nil {
		return fmt.Errorf("clone error: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("error getting worktree: %w", err)
	}

	start, err := r.ResolveRevision(plumbing.Revision("origin/" + opts.Branch))
	if err != nil {
		return fmt.Errorf("error getting %s: %w", opts.Branch, err)
	}

	log.Printf("todo... %v %v", start, w)

	return nil
}
