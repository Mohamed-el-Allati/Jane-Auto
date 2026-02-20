package web

import (
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"janeauto/models"
	"janeauto/db"
	"janeauto/jane"
	"janeauto/attestor"
)

func HomeHandler(c echo.Context) error {
	// Fetches all policies to compute the stats
	policies, err := db.GetAllPolicies()
	if err != nil {
		// If DB error, still show page but with zero stats
		policies = []models.Policy{}
	}

	policyCount := len(policies)
	attestationCount := 0
	ruleSet := make(map[string]struct{}) // for unique rule names

	for _, p := range policies {
		attestationCount += len(p.Attestations)
		for _, att := range p.Attestations {
			for _, r := range att.Rules {
				ruleSet[r.Name] = struct{}{}
			}
		}
	}
	ruleCount := len(ruleSet)

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>JANE Auto - Home</title>
	<style>
		* { margin: 0; padding: 0; box-sizing: border-box; font-family: system-ui, -apple-system, 'Segoe UI', Roboto, sans-serif; }
		body { background: #f4f6f9; display: flex; justify-content: center; align-items: center; min-height: 100vh; padding: 20px; }
		.container { max-width: 900px; width: 100%%; }
		h1 { color: #1e293b; margin-bottom: 0.5rem; font-weight: 600; }
		.subtitle { color: #475569; margin-bottom: 2rem; font-size: 1.1rem; }
		.stats-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 20px; margin-bottom: 40px; }
		.stat-card { background: white; border-radius: 16px; padding: 24px; box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1); transition: transform 0.2s; }
		.stat-card:hover { transform: translateY(-2px); box-shadow: 0 10px 15px -3px rgb(0 0 0 / 0.1); }
		.stat-number { font-size: 2.5rem; font-weight: 700; color: #0f172a; line-height: 1.2; }
		.stat-label { color: #64748b; text-transform: uppercase; letter-spacing: 0.05em; font-size: 0.875rem; margin-top: 8px; }
		.btn { display: inline-block; background: #2563eb; color: white; padding: 14px 28px; border-radius: 40px; text-decoration: none; font-weight: 500; font-size: 1.125rem; border: none; cursor: pointer; box-shadow: 0 4px 6px -1px rgba(37, 99, 235, 0.3); transition: background 0.2s, transform 0.1s; }
		.btn:hover { background: #1d4ed8; transform: scale(1.02); }
		.btn:active { transform: scale(0.98); }
		.footer { margin-top: 40px; color: #64748b; font-size: 0.875rem; text-align: center; }
		hr { border: none; border-top: 1px solid #e2e8f0; margin: 30px 0; }
	</style>
</head>
<body>
	<div class="container">
		<h1> Jane Attestation Automation</h1>
		<div class="subtitle">Your policy-driven attestation orchestrator</div>

		<div class="stats-grid">
			<div class="stat-card">
				<div class="stat-number">%d</div>
				<div class="stat-label">Policies</div>
			</div>
			<div class="stat-card">
				<div class="stat-number">%d</div>
				<div class="stat-label">Attestations</div>
			</div>
			<div class="stat-card">
				<div class="stat-number">%d</div>
				<div class="stat-label">Distinct Rules</div>
			</div>
		</div>

		<a href="/attest" class="btn"> Start New Attest</a>

		<hr>
		<div class="footer">
			<p>Powered by JANE Attestation Engine * <a href="/policies" style="color:#2563eb;">View all policies</a></p>
		</div>
	</div>
</body>
</html>`, policyCount, attestationCount, ruleCount)

	return c.HTML(http.StatusOK, html)
}

func AttestFormHandler(c echo.Context) error {
	policies, err := db.GetAllPolicies()
	fmt.Printf("[DEBUG] attestFormHandler: retrieved %d policies, error: %v\n", len(policies), err)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Failed to load policies")
	}
	if len(policies) == 0 {
		return c.String(http.StatusOK, "No policies found in database.")
	}

	var options strings.Builder
	for _, p := range policies {
		options.WriteString(fmt.Sprintf(`<option value="%s">%s</option>`, p.Name, p.Name))
	}
	fmt.Printf("[DEBUG] Options HTML: %s\n", options.String())

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>Select Policy</title>
	<style>
		* { margin: 0; padding: 0; box-sizing: border-box; font-family: system-ui, sans-serif; }
		body { background: #f4f6f9; display: flex; justify-content: center; align-items: center; min-height: 100vh; padding: 20px; }
		.card { background: white; border-radius: 24px; padding: 40px; box-shadow: 0 20px 25px -5px rgb(0 0 0 / 0.1); max-width: 500px; width: 100%%; }
		h2 { color: #1e293b; margin-bottom: 24px; font-weight: 600; }
		label { display: block; margin-bottom: 8px; color: #475569; font-weight: 500; }
		select { width: 100%%; padding: 12px 16px; border: 1px solid #cbd5e1; border-radius: 12px; font-size: 1rem; margin-bottom: 24px; background: white; cursor: pointer;  }
		select:focus { outline: 2px solid #2563eb; border-control: transparent; }
		.btn { background: #2563eb; color: white; padding: 12px 24px; border: none; border-radius: 4 0px; font-size: 1rem; font-weight; 500; cursor: pointer; width: 100%%; transition: background 0.2s; }
		.btn:hover { background: #1d4ed8; }
		.back-link { display: block; margin-top: 24px; text-align: center; color: #64748b; text-decoration: none; }
		.back-link:hover { color: #2563eb; }
	</style>
</head>
<body>
	<div class="card">
		<h2> Choose a Policy</h2>
		<form action="/attest/run" method="POST">
			<label for="policy">Policy:</label>
			<select name="policy" id="policy" required>
				<option value="" disabled selected>-- Select a policy --</option>
				%s
			</select>
			<button type="submit" class="btn"> Execute</button>
		</form>
		<a href= "/" class="back-link"> Back to Home</a>
	</div>
</body>
</html>`, options.String())

	return c.HTML(http.StatusOK, html)
}

// Executes the selected policy and displays the results
func AttestRunHandler(c echo.Context) error {
	policyName := c.FormValue("policy")
	if policyName == "" {
		return c.String(http.StatusBadRequest, "No policy selected")
	}

	// Loads policy from DB
	policy, err := db.GetPolicyByName(policyName)
	if err != nil {
		return c.String(http.StatusNotFound, "Policy not found")
	}

	// Executes the policy
	results, sessionID, err := attestor.ExecutePolicy(policy)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Execution failed: "+err.Error())
	}

	// Builds the JANE session URL
	sessionURL := buildSessionURL(policy.Jane, sessionID)

	// Current timestamp
	timestamp := time.Now().Format("02-01-2006 15:04:05")

	// Build results cards
	var cards strings.Builder
	for _, r := range results {
		// Determines card color class
		var cardClass string
		if r.Passed {
			cardClass = "pass"
		} else {
			cardClass = "fail"
		}

		// Element display: use name if available, otherwise uses eid
		elementDisplay := r.ElementID
		if r.ElementName != "" {
			elementDisplay = r.ElementName
		}

		// Builds rule details
		var ruleDetails strings.Builder
		if len(r.RuleResults) == 0 {
			ruleDetails.WriteString("<p>No rules executed for this attestation.</p>")
		} else {
			ruleDetails.WriteString("<table class='rule-table'><tr><th>Rule</th><th>Result ID</th><th>Passed</th></tr>")
			for _, ruleRes := range r.RuleResults {
				ruleName, _ := ruleRes["rule"]. (string)
				resultID, _ := ruleRes["result_id"].(string)
				passed, _ := ruleRes["passed"].(bool)
				passedStr := "Fail"
				if passed {
					passedStr = "Pass"
				}
				// Truncate result ID
				if len(resultID) > 8 {
					resultID = resultID[:8] + "..."
				}
				ruleDetails.WriteString(fmt.Sprintf("<tr><td>%s</td><td title='%s'>%s</td><td>%s</td></tr>", ruleName, resultID, resultID, passedStr))
			}
			ruleDetails.WriteString("</table>")
		}

		// Builds the HTML table
		card := fmt.Sprintf(`
		<div class="result-card %s">
			<div class="card-summary">
				<span class="element">%s</span>
				<span class="intent">%s</span>
				<span class="passed-badge">%s</span>
				<span class="claim-id" title="%s">Claim: %s</span>
			</div>
			<details class="card-details">
				<summary>Show rule details</summary>
				<div class="details-content">
					%s
				</div>
			</details>
		</div>`, cardClass, elementDisplay, r.Intent,
			map[bool]string{true: "Pass", false: "Fail"}[r.Passed],
			r.ClaimID, truncate(r.ClaimID, 8),
			ruleDetails.String())
		cards.WriteString(card)
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
	<title>Attestation Results: %s</title>
	<style>
		* { margin: 0; padding: 0; box-sizing: border-box; font-family: system-ui, sans-serif; }
		body { background: #f4f6f9; padding: 40px 20px; }
		.container { max-width: 1200px; margin: 0 auto; background: white; border-radius: 24px; padding: 32px; box-shadow: 0 20px 25px -5px rgba(0,0,0,0.1); }
		h2 { color: #1e293b; margin-bottom: 8px; }
		.policy-name { color: #2563eb; font-weight: 500; margin-bottom: 8px; }
		.session-info { margin-bottom: 24px; font-size: 0.9rem; color: #475569; }
		.session-info a { color: #2563eb; text-decoration: none; }
		.session-info a:hover { text-decoration: underline; }
		.timestamp { color: #64748b; font-size: 0.9rem; margin-bottom: 24px; }
		.results-grid { display: flex; flex-direction: column; gap: 16px; margin-top: 24px; }
		.result-card { border-radius: 12px; padding: 16px; box-shadow: 0 2px 5px rgba(0,0,0,0.05); transition: all 0.2s; }
		.result-card.pass { background-color: #f0fdf4; border-left: 6px solid #22c55e; }
		.result-card.fail { background-color: #fef2f2; border-left: 6px solid #ef4444; color: #7f1d1d; }
		.result-card.neutral { background-color: f2fce8; border-left: 6px solid #eab308; }
		.card-summary { display: flex; flex-wrap: wrap; align-items: center; gap: 16px; font-size: 1rem; }
		.element { font-weight: 600; min-width: 150px; }
		.intent { font-family: monospace; background: rgba(0,0,0,0.05); padding: 4px 8px; border-radius: 20px; }
		.passed-badge { font-weight: 500; }
		.claim-id { color: #475569; font-size: 0.9rem; margin-left: auto; }
		.card-details { margin-top: 16px; }
		.card-details summary {cursor: pointer; color: #2563eb; font-weight: 500; }
		.details-content { margin-top: 12px; padding: 12px; background: white; border-radius: 8px; border: 1px solid #e2e8f0; }
		.rule-table { width: 100%%; border-collapse: collapse; font-size: 0.9rem; }
		.rule-table th { text-align: left; padding: 8px; background: #f8fafc; border-bottom: 2px solid #cbd5e1; }
		.rule-table td { padding: 8px; border-bottom: 1px solid #e2e8f0; }
		.btn-secondary { display: inline-block; background: #f1f5f9; color: #334155; padding: 10px 20px; border-radius: 40px; text-decoration: none; font-weight: 500; margin-top: 24px; border: 1px solid #cbd5e1; transition: background 0.2s; }
		.btn-secondary:hover { background: #e2e8f0; }
		.btn-secondary + .btn-secondary { margin-left: 12px; }
	</style>
</head>
<body>
	<div class="container">
		<h2> Attestation Results: %s</h2>
		<div class="policy-name">Policy: %s</div>
		<div class="session-info">Session: <a href="%s" target= "_blank">%s</a></div>
		<div class="timestamp">Executed on: %s</div>

		<div class="results-grid">
			%s
		</div>

		<a href="/attest" class="btn-secondary"> Run another policy</a>
		<a href="/" class="btn-secondary" style="margin-left: 12px;"> Home</a>
	</div>
</body>
</html>`, policyName, policyName, policyName, sessionURL, sessionID, timestamp, cards.String())

	return c.HTML(http.StatusOK, html)
}

// Helper to truncate strings
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func PoliciesHandler(c echo.Context) error {
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

func ExecutePolicyHandler(c echo.Context) error {
	policyName := c.Param("policyName")
	fmt.Printf("\n=== STARTING EXECUTE POLICY HANDLER: %s ===\n", policyName)

	// loads policy from database
	policy, err := db.GetPolicyByName(policyName)
	if err != nil {
		return c.String(http.StatusNotFound, "Policy not found")
	}

	results, _, err := attestor.ExecutePolicy(policy)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Execution failed: "+err.Error())
	}

	// returns JSON response
	return c.JSON(http.StatusOK, map[string]interface{}{
		"results": results,
		"count:":  len(results),
	})
}

// This function constructs the JANE web UI session URL
// it takes the API base url and the session ID, and returns a URL pointing to the UI on port 8540
func buildSessionURL(apiURL, sessionID string) string {
	u, err := url.Parse(apiURL)
	if err != nil {
		// fallback, just append
		return apiURL + "/session/" + sessionID
	}

	// Replaces port with 8540
	hostParts := strings.Split(u.Host, ":")
	if len(hostParts) == 2 {
		u.Host = hostParts[0] + ":8540"
	} else {
		// no port specified, just add 8540
		u.Host = u.Host + ":8540"
	}
	//Ensures path ends with /session/ID
	return u.String() + "/session/" + sessionID
}

func DebugJaneHandler(c echo.Context) error {
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

func DebugAttestation(c echo.Context) error {
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
