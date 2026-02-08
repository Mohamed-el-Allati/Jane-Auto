package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "time"

    "janeauto/config"
)

//janeURL := config.ConfigData.Rest.Port

func janeGet(path string, target interface{}) error {
    client := &http.Client{Timeout: 10*time.Second}
    resp, err := client.Get(janeURL + path)
    if err != nil {
	return fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
	return fmt.Errorf("jane returned %s", resp.Status)
    }
    return json.NewDecoder(resp.Body).Decode(target)
}

type returnElements struct {
    Elements []string  `json:"elements"`
    Length   int       `json:"length"`
}


func janeGetElementsByName(name string) ([]string, error){
    janeURL := fmt.Sprintf("http://127.0.0.1:%s", config.ConfigData.Rest.Port)
    url := janeURL+"/elements/name/" + name

    fmt.Printf("[DEBUG] getting elements from URL: %s\n", url)

    client := &http.Client{Timeout: 10 * time.Second}
    resp, err := client.Get(url)
    if err != nil {
	return nil, fmt.Errorf("failed to get elements: %v", err)
    }
    defer resp.Body.Close()

    fmt.Printf(" body is %v\n",resp.Body)

    if resp.StatusCode != 200 {
	body, _ := ioutil.ReadAll(resp.Body)
	return nil, fmt.Errorf("JANE returned status %d: %s", resp.StatusCode, string(body))
    }

    var result struct {
	Elements []string	`json:"elements"`
	Length	 int		`json:"length"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
	return nil, fmt.Errorf("Failed to decode response: %v", err)
    }
 
    fmt.Printf("[DEBUG] Found %d elements for name '%s': %v\n", result.Length, name, result.Elements)
    return result.Elements, nil
}
