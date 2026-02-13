package jane

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// GetElementsByName retrieves element uuids by their name
func GetElementsByName(janeURL, name string) ([]string, error) {
	url := janeURL + "/elements/name/" + name
	fmt.Printf("[DEBUG] getting elements from URL: %s\n", url)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get elements: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("JANE returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Elements	[]string	`json:"elements"`
		Length		int		`json:"length"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	fmt.Printf("[DEBUG] Found %d elements for name '%s': %v\n", result.Length, name, result.Elements)
	return result.Elements, nil
}

// GetIntentItemID returns the itemid for a given intent name
func GetIntentItemID(janeURL, intentName string) (string, error) {
	// tries by name
	url := fmt.Sprintf("%s/intents/name/%s", janeURL, intentName)
	fmt.Printf("[DEBUG-INTENT] Trying endpoint: %s\n", url)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("[DEBUG-INTENT] Response status &d, body: %s\n", resp.StatusCode, string(body))

	if resp.StatusCode == 200 {
		var result struct {
			Intents	[]string	`json:"intents"`
			Length	int		`json:"length"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return "", fmt.Errorf("Failed to decode response: %v", err)
		}
		if result.Length > 0 {
			return result.Intents[0], nil
		}
	}

	// Fallback which treats intentName as ItemID
	testURL := fmt.Sprintf("%s/intent/%s", janeURL, intentName)
	fmt.Printf("[DEBUG-INTENT] Trying direct ItemID fetch: %s\n", testURL)

	testResp, err := http.Get(testURL)
	if err != nil {
		return "", fmt.Errorf("direct fetch failed: %v", err)
	}
	defer testResp.Body.Close()

	if testResp.StatusCode == 200 {
		fmt.Printf("[DEBUG-INTENT] intentName '%s' is the ItemID\n", intentName)
		return intentName, nil
	}

	return "", fmt.Errorf("intent '%s' not found by name or as ItemID", intentName)
}

// RunVerification executes a rule on a claim and returns the result ID and pass or fail
func RunVerification(janeURL, claimID, ruleName, sessionID string) (string, bool, error) {
	verifyData := map[string]interface{}{
		"cid":		claimID,
		"rule":		ruleName,
		"sid":		sessionID,
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

	passed := (result.Result == 0)
	return result.ItemID, passed, nil
}

// RunAttestation sends an attestation request and returns the claimID
func RunAttestation(janeURL, elementID, pid, endpoint, sessionID string) (string, error) {
	attestData := map[string]interface{}{
		"eid":		elementID,
		"pid":		pid,
		"epn":		endpoint,
		"sid":		sessionID,
		"parameters":	map[string]interface{}{},
	}

	body, _ := json.Marshal(attestData)
	fmt.Printf("[DEBUG] Sending attestation request:\n%s\n", string(body))

	resp, err := http.Post(fmt.Sprintf("%s/attest", janeURL), "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("attest call failed: %v", err)
	}
	defer resp.Body.Close()

	rawBody, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("[DEBUG] Attest response status: %d:\n%s\n", resp.StatusCode, string(rawBody))

	var result struct {
		ItemID	string	`json:"itemid"`
		Error	string	`json:"error"`
	}
	if err := json.Unmarshal(rawBody, &result); err != nil {
		return "", fmt.Errorf("Failed to parse attest response: %v", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("JANE error: %s", result.Error)
	}
	return result.ItemID, nil
}

// GetClaim retrieves a claim by its ID
func GetClaim(janeURL, claimID string) (map[string]interface{}, error) {
	endpoints := []string{
		fmt.Sprintf("%s/claim/%s", janeURL, claimID),
		fmt.Sprintf("%s/claims/%s", janeURL, claimID),
	}
	for _, url := range endpoints {
		fmt.Printf("[DEBUG] Trying claim endpoint: %s\n", url)

		maxAttempts := 60
		for attempt := 1; attempt <= maxAttempts; attempt++ {
			resp, err := http.Get(url)
			if err != nil {
				return nil, fmt.Errorf("failed to get claim: %v", err)
			}
			defer resp.Body.Close()

			fmt.Printf("[DEBUG] Attempt %d: Status %d\n", attempt, resp.StatusCode)

			if resp.StatusCode == 200 {
				var claim map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&claim); err != nil {
					return nil, fmt.Errorf("failed to decode claim: %v", err)
				}
				fmt.Errorf("[DEBUG] Successfully retrieved claim from %s!\n", url)
				return claim, nil
			} else if resp.StatusCode == 404 {
				time.Sleep(100 * time.Millisecond)
				continue
			} else {
				break
			}
		}
	}
	return nil, fmt.Errorf("claim %s was not found after trying all endpoints", claimID)
}

// CreateSession creates a new JANE session and returns its ID
func CreateSession(janeURL string) (string, error) {
	resp, err := http.Post(janeURL+"/session", "application/json", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	defer resp.Body.Close()

	var res struct {
		ItemID	string	`json:"itemid"`
		Error	string	`json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("failed to decode session response: %v", err)
	}
	if res.Error != "" {
		return "", fmt.Errorf("session error: %s", res.Error)
	}
	return res.ItemID, nil
}

// CloseSession deletes a JANE session
func CloseSession(janeURL, sessionID string) {
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/session/%s", janeURL, sessionID), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Failed to close JANE session:", err)
	} else {
		resp.Body.Close()
	}
}
