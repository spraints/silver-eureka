package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

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

	start, err := r.ResolveRevision(plumbing.Revision("origin/" + opts.Branch))
	if err != nil {
		if err != plumbing.ErrReferenceNotFound {
			return fmt.Errorf("error getting %s: %w", opts.Branch, err)
		}
		head, err := r.Head()
		if err != nil {
			return fmt.Errorf("error getting HEAD: %w", err)
		}
		hh := head.Hash()
		start = &hh
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("error getting worktree: %w", err)
	}

	log.Printf("creating branch %s from %s", opts.Branch, start)
	if err := w.Checkout(&git.CheckoutOptions{
		Hash:   *start,
		Branch: plumbing.ReferenceName(opts.Branch),
		Create: true,
	}); err != nil {
		return fmt.Errorf("error creating branch: %w", err)
	}

	newfile := filepath.Join(repoPath, "tick.txt")
	if err := os.WriteFile(newfile, []byte(fmt.Sprintf("%d\n", time.Now().Unix())), 0644); err != nil {
		return fmt.Errorf("error creating tick.txt")
	}

	if _, err := w.Add("tick.txt"); err != nil {
		return fmt.Errorf("git add tick.txt: %w", err)
	}

	commitHash, err := w.Commit("tick tick!", &git.CommitOptions{})
	if err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	commit, err := r.CommitObject(commitHash)
	if err != nil {
		return fmt.Errorf("getting new commit info: %w", err)
	}
	log.Printf("created commit:\n%v", commit)

	log.Print("pushing")
	if err := r.Push(&git.PushOptions{
		Auth: &authy{opts},
	}); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	log.Print("DONE!")
	return nil
}

type authy struct {
	opts options
}

func (authy) String() string {
	return "borb"
}

func (authy) Name() string {
	return "spraints-authy-borb"
}
