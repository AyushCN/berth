package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func getProjectID(client *http.Client, base string) (string, error) {
	req, _ := http.NewRequest("GET", base+"/api/projects", nil)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Projects []struct {
			ID string `json:"id"`
		} `json:"projects"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if len(result.Projects) == 0 {
		return "", fmt.Errorf("no projects found")
	}
	return result.Projects[0].ID, nil
}
