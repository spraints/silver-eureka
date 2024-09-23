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
	// GitHub API endpoint where we will post a bunch of blobs.
	createBlobURL = "https://api.github.com/repos/spraints/silver-eureka/git/blobs"

	// GitHub API endpoint where we will post a bunch of trees.
	createTreeURL = "https://api.github.com/repos/spraints/silver-eureka/git/trees"

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

	fmt.Printf("posting %d new objects...\n", blobsPerTree)
	now := time.Now()
	for i := 0; i < treeCount*blobsPerTree; i++ {
		wg.Add(1)
		go post(ch, &wg, now, i, token)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()

	var oids []string
	for oid := range ch {
		oids = append(oids, oid)
		if len(oids) == blobsPerTree {
			postTree(oids, token)
			oids = nil
		}
	}
	if len(oids) > 0 {
		postTree(oids, token)
	}
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

	for i, oid := range oids {
		reqBody.Entries = append(reqBody.Entries, treeEntry{
			fmt.Sprintf("file-%d.txt", i),
			"100644",
			"blob",
			oid,
		})
	}

	fmt.Println("todo")
}

func post(oids chan string, wg *sync.WaitGroup, now time.Time, i int, token string) {
	defer wg.Done()

	var reqBody struct {
		Content string `json:"content"`
	}
	var respBody struct {
		OID string `json:"sha"`
	}

	reqBody.Content = fmt.Sprintf("%v %v\n", i, now)

	postBody, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("%d: %v\n", i, err)
		return
	}

	req, err := http.NewRequest("POST", createBlobURL, bytes.NewReader(postBody))
	if err != nil {
		fmt.Printf("%d: %v\n", i, err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("%d: %s: %v\n", i, createBlobURL, err)
		return
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%d: error reading response: %v\n", i, err)
		return
	}

	if resp.StatusCode != 201 {
		fmt.Printf("%d: HTTP %s response to %s.\n%s\n", i, resp.Status, createBlobURL, data)
		return
	}

	if err := json.Unmarshal(data, &respBody); err != nil {
		fmt.Printf("%d: error parsing response body: %v\n%s\n", i, err, data)
		return
	}

	oids <- respBody.OID
}
