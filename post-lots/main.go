package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

const (
	githubAPIVersion = "2022-11-28"

	// GitHub API endpoint where we will post a bunch of blobs.
	createBlobURL = "https://api.github.com/repos/spraints/silver-eureka/git/blobs"

	// GitHub API endpoint where we will post a bunch of trees.
	createTreeURL = "https://api.github.com/repos/spraints/silver-eureka/git/trees"

	// GitHub API endpoint where we will post a bunch of commits.
	createCommitURL = "https://api.github.com/repos/spraints/silver-eureka/git/commits"

	// How many trees to create.
	treeCount = 10

	// How many blobs to create in each tree.
	blobsPerTree = 10
)

func main() {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Printf("GITHUB_TOKEN must be set.\n")
		return
	}

	var wg sync.WaitGroup
	ch := make(chan string)

	totalBlobs := treeCount * blobsPerTree
	fmt.Printf("posting %d new objects...\n", totalBlobs)
	now := time.Now()
	for i := 0; i < totalBlobs; i++ {
		wg.Add(1)
		go postBlob(ch, &wg, now, i, token)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()

	var (
		blobCount = 0
		treeCount = 0
	)

	var oids []string
	for oid := range ch {
		blobCount += 1
		oids = append(oids, oid)
		if len(oids) == blobsPerTree {
			treeCount += 1
			postTree(oids, token)
			oids = nil
		}
	}
	if len(oids) > 0 {
		treeCount += 1
		postTree(oids, token)
	}

	fmt.Printf("created %d blobs, %d trees, %d commits\n", blobCount, treeCount, treeCount)
}

func postCommit(treeOID string, token string) {
	var reqBody struct {
		Message string `json:"message"`
		Tree    string `json:"tree"`
	}
	type commit struct {
		OID string `json:"sha"`
		URL string `json:"html_url"`
	}
	var respBody commit

	reqBody.Message = "test commit"
	reqBody.Tree = treeOID

	if err := doRequest(createCommitURL, &reqBody, &respBody, token); err != nil {
		fmt.Printf("create commit error: %v\n", err)
		return
	}

	fmt.Printf("created commit %#v\n", respBody)
}

func postTree(oids []string, token string) {
	type treeEntry struct {
		Path string `json:"path"`
		Mode string `json:"mode"`
		Type string `json:"type"`
		OID  string `json:"sha"`
	}
	var reqBody struct {
		Entries []treeEntry `json:"tree"`
	}
	var respBody struct {
		OID string `json:"sha"`
	}

	for i, oid := range oids {
		reqBody.Entries = append(reqBody.Entries, treeEntry{
			fmt.Sprintf("file-%d.txt", i),
			"100644",
			"blob",
			oid,
		})
	}

	if err := doRequest(createTreeURL, &reqBody, &respBody, token); err != nil {
		fmt.Printf("create tree error: %v\n", err)
		return
	}

	fmt.Printf("created tree %s\n", respBody.OID)
	postCommit(respBody.OID, token)
}

func postBlob(oids chan string, wg *sync.WaitGroup, now time.Time, i int, token string) {
	defer wg.Done()

	var reqBody struct {
		Content string `json:"content"`
	}
	var respBody struct {
		OID string `json:"sha"`
	}

	reqBody.Content = fmt.Sprintf("%v %v\n", i, now)

	if err := doRequest(createBlobURL, &reqBody, &respBody, token); err != nil {
		fmt.Printf("[%d] error: %v\n", i, err)
		return
	}

	oids <- respBody.OID
}

func doRequest(url string, reqBody, respBody interface{}, token string) error {
	postBody, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(postBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", githubAPIVersion)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != 201 {
		return fmt.Errorf("error HTTP %s response to %s\nresponse body:\n%s", resp.Status, url, data)
	}

	if err := json.Unmarshal(data, &respBody); err != nil {
		return fmt.Errorf("error parsing response body: %w\n%s", err, data)
	}

	return nil
}
