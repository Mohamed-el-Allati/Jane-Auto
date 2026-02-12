package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

func homeHandler(c echo.Context) error {
	html := `
    <!DOCTYPE html>
    <html>
    <head><title>Jane Auto Test</title></head>
    <body>
	<h1>press button</h1>
	<form action="/policies" method="get">
	    <button style="font-size:18px;padding:8px 16px;">Connected! Show Policies</button>
	</form>
    </body>
    </html>
    `
	return c.HTML(http.StatusOK, html)
}

func policiesHandler(c echo.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collection := client.Database("testdb").Collection("policies")
	cursor, err := collection.Find(ctx, map[string]interface{}{})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error retrieving policies")
	}
	defer cursor.Close(ctx)

	cursor, err = collection.Find(ctx, map[string]interface{}{})
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error retrieving policies for decoding")
	}

	var policies []Policy
	if err = cursor.All(ctx, &policies); err != nil {
		return c.String(http.StatusInternalServerError, "Error decoding policies: "+err.Error())
	}

	var html strings.Builder
	html.WriteString("<!DOCTYPE html><html><head><title>Policies</title></head><body>")
	html.WriteString("<h1>All Policies</h1>")

	for _, policy := range policies {
		html.WriteString(fmt.Sprintf(`
	    <h2>%s</h2>
	    <p><b>Description:</b> %s</p>
	    <p><b>Jane:</b> %s</p>
	    <h3>Collection</h3>
	    <p><b>Items:</b> %v</p>
	    <p><b>Tags:</b> %v</p>
	    <p><b>Names:</b> %v</p>
	    <form action="/execute/%s" method="post">
		<button type="submit">Execute</button>
	    </form>
	    <hr>
	`,
			policy.Name,
			policy.Description,
			policy.Jane,
			strings.Join(policy.Collection.Items, ", "),
			strings.Join(policy.Collection.Tags, ", "),
			strings.Join(policy.Collection.Names, ", "),
			policy.Name,
		))
	}

	html.WriteString(`<p><a href="/">Back to home</a></p></body></html>`)
	return c.HTML(http.StatusOK, html.String())
}

type AttestationResult struct {
	ElementID   string                   `bson:"element_id" json:"element_id"`
	Intent      string                   `bson:"intent" json:"intent"`
	Claim       interface{}              `bson:"claim" json:"claim"`
	Passed      bool                     `bson:"passed" json:"passed"`
	RuleResults []map[string]interface{} `bson:"rule_results" json:"rule_results"`
	ClaimID     string                   `bson:"claim_id" json:"claim_id"`
}

func runRules(janeURL, claimID, sessionID string, rules []Rule) (bool, []map[string]interface{}) {
	allPassed := true
	ruleResults := []map[string]interface{}{}

	for _, rule := range rules {
		fmt.Printf("[DEBUG] Running rule: %s on claim %s\n", rule.Name, claimID)

		// This calls the function "janeRunVerification" in janeRest.go
		resultID, passed, err := janeRunVerification(janeURL, claimID, rule.Name, sessionID)
		if err != nil {
			fmt.Printf("[ERROR] Failed to run rule %s: %v\n", rule.Name, err)
			ruleResults = append(ruleResults, map[string]interface{}{
				"rule":   rule.Name,
				"passed": false,
				"error":  err.Error(),
			})
			allPassed = false
			continue
		}

		ruleResults = append(ruleResults, map[string]interface{}{
			"rule":      rule.Name,
			"passed":    passed,
			"result_id": resultID,
		})

		if !passed {
			allPassed = false
		}
	}

	return allPassed, ruleResults
}

func unique(items []string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, i := range items {
		if _, ok := seen[i]; !ok {
			seen[i] = struct{}{}
			result = append(result, i)
		}
	}
	return result
}

func createJaneSession(janeURL string) (string, error) {
	resp, err := http.Post(janeURL+"/session", "application/json", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	defer resp.Body.Close()

	var res struct {
		ItemID string `json:"itemid"`
		Error  string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("failed to decode session response: %v", err)
	}
	if res.Error != "" {
		return "", fmt.Errorf("session error: %s", res.Error)
	}
	return res.ItemID, nil
}

func closeJaneSession(janeURL, sid string) {
	req, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("%s/session/%s", janeURL, sid), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Failed to close JANE session:", err)
	} else {
		resp.Body.Close()
	}
}

func executePolicyHandler(c echo.Context) error {
	policyName := c.Param("policyName")
	fmt.Printf("\n=== STARTING EXECUTE POLICY HANDLER: %s ===\n", policyName)

	// loads policy from database
	policy, err := dbGetPolicyByName(policyName)
	if err != nil {
		return c.String(http.StatusNotFound, "Policy not found")
	}

	results, err := ExecutePolicy(policy)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Execution failed: "+err.Error())
	}

	// returns JSON response
	return c.JSON(http.StatusOK, map[string]interface{}{
		"results": results,
		"count:":  len(results),
	})
}

func janeRunAttestation(janeURL, elementid, pid, endpoint, sid string) (string, error) {
	attestData := map[string]interface{}{
		"eid":        elementid,
		"pid":        pid,
		"epn":        endpoint,
		"sid":        sid,
		"parameters": map[string]interface{}{},
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
		ItemID string `json:"itemid"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(rawBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse attest response: %v", err)
	}
	if result.Error != "" {
		return "", fmt.Errorf("JANE error: %s", result.Error)
	}
	claimID := result.ItemID

	if strings.Contains(strings.ToLower(claimID), "error") {
		return "", fmt.Errorf("JANE error: %s", claimID)
	}

	return claimID, nil
}

func janeGetClaim(janeURL, claimID string) (map[string]interface{}, error) {
	endpoints := []string{
		fmt.Sprintf("%s/claim/%s", janeURL, claimID),
		fmt.Sprintf("%s/claims/%s", janeURL, claimID),
	}
	for _, url := range endpoints {
		fmt.Printf("[DEBUG] Trying claim endpoint: %s\n", url)

		maxAttempts := 60 // 6 second grace
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
					return nil, fmt.Errorf("Failed to decode claim: %v", err)
				}
				fmt.Printf("[DEBUG] Successfully retrieved claim from %s!\n", url)
				return claim, nil
			} else if resp.StatusCode == 404 {
				time.Sleep(100 * time.Millisecond)
				continue
			} else {
				// try the next endpoint
				break
			}
		}
	}
	return nil, fmt.Errorf("Claim %s was not found after trying all the endpoints", claimID)
}

func debugJaneHandler(c echo.Context) error {
	janeBaseURL := "http://localhost:8520"

	elements, err := janeGetElementsByName("bobafet")
	if err != nil {
		return c.JSON(500, map[string]interface{}{
			"error":   "Failed to get elements",
			"details": err.Error(),
		})
	}

	intentsResp, err := http.Get(janeBaseURL + "/intents")
	var intentsData interface{}
	if err == nil {
		defer intentsResp.Body.Close()
		body, _ := ioutil.ReadAll(intentsResp.Body)
		json.Unmarshal(body, &intentsData)
	}

	return c.JSON(200, map[string]interface{}{
		"bobafet_elements": elements,
		"jane_intents":     intentsData,
		"jane_url":         janeBaseURL,
	})
}

func debugAttestation(c echo.Context) error {
	janeURL := "http://localhost:8520"

	sid, err := createJaneSession(janeURL)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	attestData := map[string]interface{}{
		"eid":        "2d1e8307-3987-4bcf-a182-2b3504394a4e",
		"pid":        "std::intent::sys::info",
		"epn":        "tarzan",
		"sid":        sid,
		"parameters": map[string]interface{}{},
	}

	body, _ := json.Marshal(attestData)
	resp, err := http.Post(janeURL+"/attest", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}
	defer resp.Body.Close()

	var result struct {
		ItemID string `json:"itemid"`
		Error  string `json:"error"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	return c.JSON(200, map[string]interface{}{
		"session_id": sid,
		"claim_id":   result.ItemID,
		"status":     resp.StatusCode,
		"error":      result.Error,
	})
}
