package main

import (
    "bytes"
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
    fmt.Printf("[DEBUG-INTENT] Trying endpoint: %s\n", url)

    resp, err := http.Get(url)
    if err != nil {
	return "", fmt.Errorf("HTTP request failed: %v", err)
    }
    defer resp.Body.Close()

    body, _ := ioutil.ReadAll(resp.Body)
    fmt.Printf("[DEBUG-INTENT] Response status %d, body: %s\n", resp.StatusCode, string(body))

    if resp.StatusCode == 200 {
	var result struct {
	    Intents	[]string `json:"intents"`
	    Length	int	 `json:"length"`
	}
    	if err := json.Unmarshal(body, &result); err != nil {
	    return "", fmt.Errorf("failed to decode response: %v", err)
    	}

	if result.Length > 0 {
	    // found by name, returns the first itemID
	    return result.Intents[0], nil
	}
    }

    // if name endpoint fails, checks if intentName is the ItemID by trying to fetch it direct
    testURL := fmt.Sprintf("%s/intent/%s", janeURL, intentName)
    fmt.Printf("[DEBUG-INTENT] Trying direct ItemID fetch: %s\n", testURL)

    testResp, err := http.Get(testURL)
    if err != nil {
	return "", fmt.Errorf("direct fetch failed: %v", err)
    }
    defer testResp.Body.Close()

    if testResp.StatusCode == 200 {
	// then intentname is the itemid
	fmt.Printf("[DEBUG-INTENT] intentName '%s' is the ItemID\n", intentName)
	return intentName, nil
    }

    return "", fmt.Errorf("intent '%s' not found by name or as ItemID", intentName)
}

func janeRunVerification(janeURL, claimID, ruleName, sessionID string) (string, bool, error) {
    verifyData := map[string]interface{}{
	"cid":	claimID,
	"rule":	ruleName,
	"sid":	sessionID,
	"parameters":	map[string]interface{}{},
    }

    body, _ := json.Marshal(verifyData)
    fmt.Printf("[DEBUG] Sending verification request:\n%s\n", string(body))

    resp, err := http.Post(fmt.Sprintf("%s/verify", janeURL), "application/json", bytes.NewBuffer(body))
    if err != nil {
	return "", false, fmt.Errorf("verify call failed: %v", err)
    }
    defer resp.Body.Close()

    rawBody, _ := ioutil.ReadAll(resp.Body)
    fmt.Printf("[DEBUG] Verify response status: %d:\n%s\n", resp.StatusCode, string(rawBody))

    var result struct {
	ItemID	string	`json:"itemid"`
	Result	int	`json:"result"`
	Error	string	`json:"error"`
    }

    if err := json.Unmarshal(rawBody, &result); err != nil {
	return "", false, fmt.Errorf("failed to parse verify response: %v", err)
    }

    if result.Error != "" {
	return "", false, fmt.Errorf("JANE verification error: %s", result.Error)
    }

    //determines whether verification passed
    passed := (result.Result == 0)

    return result.ItemID, passed, nil
}

