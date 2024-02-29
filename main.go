package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/client"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

func main() {
	opts := options{
		URL:          "https://github.com/spraints/silver-eureka",
		Branch:       "testing-123",
		User:         "spraints",
		Token:        os.Getenv("GITHUB_TOKEN"),
		ShowProgress: false,
	}

	if opts.Token == "" {
		opts.Token = tryReadDotGitHubToken()
	}

	var skip int
	for i, arg := range os.Args[1:] {
		if skip > 0 {
			skip--
			continue
		}
		if arg == "-p" || arg == "--progress" {
			log.Printf("showing progress")
			opts.ShowProgress = true
		} else if arg == "-v" || arg == "--verbose" {
			log.Printf("showing progress")
			opts.ShowProgress = true
			c := http.NewClient(newVerboseHTTPClient())
			client.InstallProtocol("https", c)
			client.InstallProtocol("http", c)
		} else if arg == "-r" || arg == "--review-lab" {
			opts.URL = "https://spraints.review-lab.github.com/spraints/silver-eureka"
		} else if arg == "-g" || arg == "--garage" {
			opts.URL = "https://garage.github.com/spraints/silver-eureka"
		} else if arg == "-u" || arg == "--url" {
			opts.URL = os.Args[i+2]
			skip = 1
		} else {
			log.Fatalf("illegal arg %q", arg)
		}
	}

	log.Printf("pushing to %v", opts.URL)

	if err := mainImpl(opts); err != nil {
		log.Fatal(err)
	}
}

type options struct {
	URL          string
	Branch       string
	User         string
	Token        string
	ShowProgress bool
}

func mainImpl(opts options) error {
	tmpdir, err := os.MkdirTemp("", "silver-eureka-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpdir)

	repoPath := filepath.Join(tmpdir, "testing")

	auth := &http.BasicAuth{
		Username: opts.User,
		Password: opts.Token,
	}

	log.Printf("cloning %s to %q", opts.URL, repoPath)
	r, err := git.PlainClone(repoPath, false, &git.CloneOptions{URL: opts.URL, Auth: auth})
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
	pushOpts := &git.PushOptions{
		RefSpecs: []config.RefSpec{
			config.RefSpec(commitHash.String() + ":refs/heads/" + opts.Branch),
		},
		Auth: auth,
	}
	if opts.ShowProgress {
		pushOpts.Progress = &progress{}
	}
	if err := r.Push(pushOpts); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	log.Print("DONE!")
	return nil
}

type progress struct{}

func (progress) Write(msg []byte) (int, error) {
	if bytes.Equal(msg, []byte{0}) {
		return 1, nil
	}

	log.Printf("progress:\n%s", msg)
	return len(msg), nil
}

func tryReadDotGitHubToken() string {
	data, err := os.ReadFile(filepath.Join(os.Getenv("HOME"), ".github-token"))
	if err != nil {
		return ""
	}

	const prefix = "GITHUB_TOKEN="
	i := bytes.Index(data, []byte(prefix))
	if i == -1 {
		return ""
	}

	data = data[i+len(prefix):]

	if i := bytes.IndexRune(data, '\n'); i != -1 {
		data = data[:i]
	}

	return string(data)
}
