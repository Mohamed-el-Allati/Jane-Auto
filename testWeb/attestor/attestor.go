package attestor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"janeauto/models"
	"janeauto/jane"
)

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

func runRules(janeURL, claimID, sessionID string, rules []models.Rule) (bool, []map[string]interface{}) {
	allPassed := true
	ruleResults := []map[string]interface{}{}

	for _, rule := range rules {
		fmt.Printf("[DEBUG] Running rule: %s on claim %s\n", rule.Name, claimID)

		resultID, passed, err := jane.RunVerification(janeURL, claimID, rule.Name, sessionID)
		if err != nil {
			fmt.Printf("[ERROR] Failed to run rule %s: %v\n", rule.Name, err)
			ruleResults = append(ruleResults, map[string]interface{}{
				"rule":		rule.Name,
				"passed":	false,
				"error":	err.Error(),
			})
			allPassed = false
			continue
		}
		ruleResults = append(ruleResults, map[string]interface{}{
			"rule":		rule.Name,
			"passed":	passed,
			"result_id":	resultID,
		})

		if !passed {
			allPassed = false
		}
	}

	return allPassed, ruleResults
}

// ExecutePolicy runs the entire attestation process for any given policy.
func ExecutePolicy(policy *models.Policy) ([]models.AttestationResult, error) {
	fmt.Printf("\n=== EXECUTING POLICY: %s ===\n", policy.Name)

	janeURL := policy.Jane

	// Fetches intents from JANE
	fmt.Printf("[DEBUG] Fetching intents from: %s\n", janeURL+"/intents")
	resp, err := http.Get(janeURL + "/intents")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch intents: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

	var intentData struct {
		Intents []string `json:"intents"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&intentData); err != nil {
		return nil, fmt.Errorf("failed to decode intents: %v", err)
	}

	// builds intent name -> itemID map
	intentNameToItemID := make(map[string]string)
	for _, intentName := range intentData.Intents {
		normalizedName := strings.ReplaceAll(intentName, " ", "")
		itemID, err := jane.GetIntentItemID(janeURL, normalizedName)
		if err != nil {
			fmt.Printf("[WARNING] Could not get ItemID for intent '%s': %v\n", normalizedName, err)
		} else {
			intentNameToItemID[normalizedName] = itemID
		}
	}
	fmt.Printf("[DEBUG] Intent map has %d entries\n", len(intentNameToItemID))

	// Resolves element names to UUIDs
	var elementIDs []string
	elementIDs = append(elementIDs, policy.Collection.Items...)
	for _, name := range policy.Collection.Names {
		fmt.Printf("[DEBUG] Looking for elements with name : %s\n", name)
		ids, _ := jane.GetElementsByName(janeURL, name)
		elementIDs = append(elementIDs, ids...)
	}
	elementIDs = unique(elementIDs)

	var filtered []string
	for _, id := range elementIDs {
		if strings.TrimSpace(id) != "" {
			filtered = append(filtered, id)
		}
	}
	elementIDs = filtered
	fmt.Printf("[DEBUG] ElementIDs (filtered): %v\n", elementIDs)

	// creates the jane session
	sid, err := jane.CreateSession(janeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create JANE session: %v", err)
	}
	// ensures session is closed after we finish
	defer jane.CloseSession(janeURL, sid)

	// this is the main attestation loop
	var results []models.AttestationResult
	fmt.Printf("[DEBUG] Starting attestation loop. Elements: %d, Attestation: %d\n", len(elementIDs), len(policy.Attestations))

	for _, eid := range elementIDs {
		for _, attest := range policy.Attestations {
			normalizedPolicyIntent := strings.ReplaceAll(attest.Intent, " ", "")
			pid, ok := intentNameToItemID[normalizedPolicyIntent]

			fmt.Printf("\n[ATTESTATION] Element: %s, Intent: %s -> Found ItemID: %s\n", eid, attest.Intent, pid)

			if !ok {
				fmt.Printf("[ERROR] Intent not found on JANE: %s\n", attest.Intent)
				results = append(results, models.AttestationResult{
					ElementID: eid,
					Intent:    attest.Intent,
					Claim:     map[string]interface{}{"error": "Intent not found on JANE"},
					Passed:    false,
				})
				continue
			}

			// runs the attestation part
			claimID, err := jane.RunAttestation(janeURL, eid, pid, attest.Endpoint, sid)
			if err != nil {
				results = append(results, models.AttestationResult{
					ElementID: eid,
					Intent:    attest.Intent,
					Claim:     map[string]interface{}{"error": err.Error()},
					Passed:    false,
				})
				continue
			}

			// retrieves the claim
			claim, err := jane.GetClaim(janeURL, claimID)
			if err != nil {
				results = append(results, models.AttestationResult{
					ElementID: eid,
					Intent:    attest.Intent,
					Claim:     map[string]interface{}{"error": err.Error()},
					Passed:    false,
				})
				continue
			}

			// runs all rules for this attestation
			passed, ruleResults := runRules(janeURL, claimID, sid, attest.Rules)

			// saves the results
			results = append(results, models.AttestationResult{
				ElementID:   eid,
				Intent:      attest.Intent,
				Claim:       claim,
				Passed:      passed,
				RuleResults: ruleResults,
				ClaimID:     claimID,
			})
		}
	}

	fmt.Printf("[DEBUG] Total results: %d\n", len(results))
	return results, nil
}
