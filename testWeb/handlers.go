package main

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"io/ioutil"
	"net/http"
	"strings"

	"janeauto/models"
	"janeauto/db"
	"janeauto/jane"
	"janeauto/attestor"
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
	policies, err := db.GetAllPolicies()
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error retrieving policies: "+err.Error())
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

func runRules(janeURL, claimID, sessionID string, rules []models.Rule) (bool, []map[string]interface{}) {
	allPassed := true
	ruleResults := []map[string]interface{}{}

	for _, rule := range rules {
		fmt.Printf("[DEBUG] Running rule: %s on claim %s\n", rule.Name, claimID)

		// This calls the function "janeRunVerification" in jane.go
		resultID, passed, err := jane.RunVerification(janeURL, claimID, rule.Name, sessionID)
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

func executePolicyHandler(c echo.Context) error {
	policyName := c.Param("policyName")
	fmt.Printf("\n=== STARTING EXECUTE POLICY HANDLER: %s ===\n", policyName)

	// loads policy from database
	policy, err := db.GetPolicyByName(policyName)
	if err != nil {
		return c.String(http.StatusNotFound, "Policy not found")
	}

	results, err := attestor.ExecutePolicy(policy)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Execution failed: "+err.Error())
	}

	// returns JSON response
	return c.JSON(http.StatusOK, map[string]interface{}{
		"results": results,
		"count:":  len(results),
	})
}

func debugJaneHandler(c echo.Context) error {
	janeBaseURL := "http://localhost:8520"

	elements, err := jane.GetElementsByName(janeBaseURL,"bobafet")
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

	sid, err := jane.CreateSession(janeURL)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	claimID, err := jane.RunAttestation(janeURL, "2d1e8307-3987-4bcf-a182-2b3504394a4e", "std::intent::sys::info", "tarzan", sid)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"session_id": sid,
		"claim_id":   claimID,
	})
}
