package gitDownloader

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
)

var repo, tag, token string

func prepareRequest(url string) *http.Request {

	req, _ := http.NewRequest("GET", url, nil)

	req.Header.Add("Authorization", fmt.Sprintf("token %s", token))

	req.Header.Add("User-Agent", "metal3d-go-client")
	return req
}

// Download resource from given url, write 1 in chan when finished
func DownloadResource(id float64, c chan int, destFolder string) {
	defer func() { c <- 1 }()
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/assets/%.0f", repo, id)
	fmt.Printf("Start: %s\n", url)
	req := prepareRequest(url)

	req.Header.Add("Accept", "application/octet-stream")

	client := http.Client{}
	resp, _ := client.Do(req)

	disp := resp.Header.Get("Content-disposition")
	re := regexp.MustCompile(`filename=(.+)`)
	matches := re.FindAllStringSubmatch(disp, -1)

	if len(matches) == 0 || len(matches[0]) == 0 {
		log.Println("WTF: ", matches)
		log.Println(resp.Header)
		log.Println(req)
		return
	}

	disp = matches[0][1]

	f, err := os.OpenFile(destFolder+"/"+disp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
	if err != nil {
		log.Fatal(err)
	}

	b := make([]byte, 4096)
	var i int

	for err == nil {
		i, err = resp.Body.Read(b)
		f.Write(b[:i])
	}
	fmt.Printf("Finished: %s -> %s\n", url, disp)
	f.Close()
}

func DownloadReleaseFiles(repo1 string, tag1 string, token1 string, files []string) []interface{} {

	repo = repo1
	tag = tag1
	token = token1

	if len(repo) == 0 {
		log.Fatal("No repository provided")
	}

	// command to call
	command := "releases/latest"
	if len(tag) > 0 {
		command = fmt.Sprintf("releases/tags/%s", tag)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", repo, command)

	// create a request with basic-auth
	req := prepareRequest(url)

	// Add required headers
	req.Header.Add("Accept", "application/vnd.github.v3.text-match+json")
	req.Header.Add("Accept", "application/vnd.github.moondragon+json")

	// call github
	client := http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		log.Fatal("Error while making request", err)
	}

	// status in <200 or >299
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		log.Fatalf("Error: %d %s", resp.StatusCode, resp.Status)
	}

	bodyText, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error reading response", err)
	}

	// prepare result
	result := make(map[string]interface{})
	json.Unmarshal(bodyText, &result)

	// print download url
	results := make([]interface{}, 0)

	for _, asset := range result["assets"].([]interface{}) {
		// fmt.Printf("Name: %s", asset.(map[string]interface{})["name"])
		var test string = asset.(map[string]interface{})["name"].(string)

		if contains(files, test) {
			results = append(results, asset.(map[string]interface{})["id"])
		}

	}

	return results
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
