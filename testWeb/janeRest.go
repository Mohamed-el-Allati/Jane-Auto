package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "time"

    "janeauto/config"
)

type returnElements struct {
    Elements []string  `json:"elements"`
    Length   int       `json:"length"`
}


func janeGetElementsByName(name string) ([]string, error){
    janeBaseURL := config.ConfigData.Jane.URL
    url := janeBaseURL+"/elements/name/" + name

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

func janeGetIntentItemID(janeURL, intentName string) (string, error) {
    // Calls the rest api to get intents by name
    url := fmt.Sprintf("%s/intents/name/%s", janeURL, intentName)
    fmt.Printf("[DEBUG] Getting ItemID for intent '%s' from: %s\n", intentName, url)

    resp, err := http.Get(url)
    if err != nil {
	return "", fmt.Errorf("failed to call intents/name endpoint: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != 200 {
	body, _ := ioutil.ReadAll(resp.Body)
	return "", fmt.Errorf("endpoint returned status %d: %s", resp.StatusCode, string(body))
    }
    // Decodes the response
    var result struct {
	Intents	[]string `json:"intents"` // list of itemids for intents with this name
	Length	int	 `json:"length"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
	return "", fmt.Errorf("failed to decode response: %v", err)
    }

    // checks whether we found any intents
    if result.Length == 0 {
	return "", fmt.Errorf("no intent found with name: %s", intentName)
    }

    // assumes the first itemid in the list is the correct one
    itemID := result.Intents[0]
    fmt.Printf("[DEBUG] Mapped intent '%s' -> ItemID '%s'\n", intentName, itemID)
    return itemID, nil
}
