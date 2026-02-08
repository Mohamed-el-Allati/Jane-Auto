package main 

import (
    "bytes"
    "encoding/json"
    "context"
    "io/ioutil"
    "fmt"
    "net/http"
    "strings"
    "time"
    "github.com/labstack/echo/v4"
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
    ElementID	string		`bson:"element_id" json:"element_id"`
    Intent	string		`bson:"intent" json:"intent"`
    Claim	interface{}	`bson:"claim" json:"claim"`
    Passed	bool		`bson:"passed" json:"passed"`
}

func attestPolicyHandler(c echo.Context) error {
    policyName := c.Param("policyName")
    fmt.Printf("[attest] Start attesting policy: %s\n", policyName)

    policy, err := dbGetPolicyByName(policyName)
    if err != nil {
	fmt.Printf("[attest][ERROR] unable to fetc policy: %v\n", err)
	return c.String(http.StatusNotFound, "Policy not Found")
    }
    fmt.Printf("[attest] Loaded policy: %+v\n", policy)

    resp, err := http.Get(janeURL + "/intents")
    if err != nil {
	return c.String(http.StatusInternalServerError, "Failed to fetch intents: "+err.Error())
    }
    defer resp.Body.Close()

    var intentData struct {
	Intents []struct {
	    ItemID	string	`json:"itemid"`
	    Name	string	`json:"name"`
	}`json:"intents"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&intentData); err != nil {
	return c.String(http.StatusInternalServerError, "Failed to decode intents: "+err.Error())
    }

    fmt.Printf("[DEBUG] intents from JANE: %+v\n", intentData.Intents)

    intentNameToID := make(map[string]string)
    for _, i := range intentData.Intents {
	intentNameToID[i.Name] = i.ItemID
    }

    fmt.Printf("[DEBUG] intentNameToID = %+v\n", intentNameToID)


    sid, err := createJaneSession(policy.Jane)
    if err != nil {
	fmt.Printf("[attest][ERROR] Failed to create Jane session: %v\n", err)
	return c.String(http.StatusInternalServerError, "Failed to create Jane session")
    }
    defer closeJaneSession(policy.Jane, sid)

    var elementIDs []string
    elementIDs = append(elementIDs, policy.Collection.Items...)
    for _, name := range policy.Collection.Names {
	fmt.Printf("[attest] Getting name %v\n",name)
	ids, err := janeGetElementsByName(name)
        fmt.Printf("[attest] Returned ids for name %v: %v\n",name, ids)
	if err != nil {
            fmt.Printf("[attest][ERROR]Error returned for name %v: %v\n", name, err)
	} else {
	    elementIDs = append(elementIDs, ids...)
        }
    }

    elementIDs = unique(elementIDs)
    fmt.Printf("[attest] ElementIDs is %v\n",elementIDs)

    var results []AttestationResult

    for _, eid := range elementIDs {
	for _, attest := range policy.Attestations {

	    pid, ok := intentNameToID[attest.Intent]
	    if !ok {
		fmt.Printf("[attest][ERROR] Intent not found on Jane: %s\n", attest.Intent)
		results = append(results, AttestationResult{
		    ElementID: 	eid,
		    Intent:	attest.Intent,
		    Claim:	map[string]{"error": "Intent not found on JANE"},
		    Passed:	false,
		})
		continue
	    }
	    fmt.Printf("[attest] Running attestation eid=%s pid=%s intent=%s endpoint=%s\n",
		eid, pid, attest.Intent, attest.Endpoint,
	    )

	    claim, err := janeRunAttestation(policy.Jane, eid, pid, attest.Endpoint, sid)
	    if err != nil {
		fmt.Printf("[attest][ERROR] Attestation failed for eid=%s, intent=%s: %v\n", eid, attest.Intent, err)
		results = append(results, AttestationResult{
		    ElementID:	eid,
		    Intent:	attest.Intent,
		    Claim:	map[string]string{"error": err.Error()},
		    Passed:	false,
		})
		continue
	    }

	    fmt.Printf("[attest] Claim received for eid=%s intent=%s: %+v\n", eid, attest.Intent, claim)

	    passed := runRules(claim, attest.Rules)
	    if m, ok := claim["message"].(string); ok && m == "Not Found" {
		passed = false
	    }

	    results = append(results, AttestationResult{
		ElementID: eid,
		Intent: attest.Intent,
		Claim: claim,
		Passed: passed,
	    })
	}
    }
    fmt.Printf("[attest] Completed attesting for policy: %s\n", policyName)
    return c.JSON(http.StatusOK, results)
    fmt.Printf("[attest] Final results: %+v\n", results)
}

fmt.Printf("[attest] Final results: %+v\n", results)

func runRules(claim map[string]interface{}, rules []Rule) bool {
    if msg, ok := claim["error"].(string); ok && msg != "" {
	return false
    }
    if msg, ok := claim["message"].(string); ok && (msg == "Not Found" || strings.Contains(strings.ToLower(msg), "error")) {
	return false
    }
    return true
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
    fmt.Printf("\n=== STARTING EXECUTE POLICY: %s ===\n", policyName)

    policy, err := dbGetPolicyByName(policyName)
    if err != nil {
	return c.String(http.StatusNotFound, "Policy not found")
    }

    fmt.Printf("[DEBUG] Policy loaded: %s\n", policy.Name)
    fmt.Printf("[DEBUG] Jane URL from policy: %s\n", policy.Jane)
    fmt.Printf("[DEBUG] Number of attestations: %d\n", len(policy.Attestations))

    janeURL := policy.Jane
   
    resp, err := http.Get(janeURL + "/intents")
    if err != nil {
	fmt.Printf("[ERROR] Failed to fetch intents: %v\n", err)
	return c.String(http.StatusInternalServerError, "Failed to fetch intents: "+err.Error())
    }
    defer resp.Body.Close()
    
    fmt.Printf("[DEBUG] intentNameToID = %+v\n", intentNameToID)
    fmt.Printf("[DEBUG] Number of intents from JANE: %d\n", len(intentData.Intents))
    fmt.Printf("[DEBUG] Intent Map: %+v\n", intentNameToID)

    bodyBytes, _ := ioutil.ReadAll(resp.Body)
    fmt.Printf("[DEBUG] Raw intents response from JANE (status %d):\n%s\n", resp.StatusCode,  string(bodyBytes))

    resp.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))

   // var intentData struct {
//	Intents []string `bson:"intents" json:"intents"`
  //  }

    var intentData struct {
	Intents []struct {
	    ItemID 	string 	`json:"itemid"`
	    Name	string	`json:"name"`
	}`json:"intents"`
    }

    if err := json.NewDecoder(resp.Body).Decode(&intentData); err != nil {
	return c.String(http.StatusInternalServerError, "Failed to decode intents: "+err.Error())
    }

    fmt.Printf("[DEBUG] intents from JANE: %v\n", intentData.Intents)

    //validIntents := make(map[string]struct{})
    //for _, i := range intentData.Intents {
	//validIntents[i] = struct{}{}
    //}
 
    intentNameToID := make(map[string]string)
    for _, i := range intentData.Intents {
	intentNameToID[i.Name] = i.ItemID
    }

    var elementIDs []string
    elementIDs = append(elementIDs, policy.Collection.Items...)
    for _, name := range policy.Collection.Names {
	ids, _ := janeGetElementsByName(name)
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
    fmt.Printf("[attest] ElementIDs (filtered) is %v\n", elementIDs)

    sid, err := createJaneSession(janeURL)
    if err != nil {
	return c.String(http.StatusInternalServerError, "Failed to create a Jane session: "+err.Error())
    }
    defer closeJaneSession(janeURL, sid)

    var results []AttestationResult

    fmt.Printf("[DEBUG] Starting attestation loop. Elements: %d, Attestations: %d\n", len(elementIDs), len(policy.Attestations))

    for _, eid := range elementIDs {
	for _, attest := range policy.Attestations {
	    pid, ok := intentNameToID[attest.Intent]
	    fmt.Printf("\n[ATTESTATION] Element: %s, Intent: %s, Found in map: %v, PID: %s\n", eid, attest.Intent, ok, pid)
	    if !ok {
		fmt.Printf("[ERROR] Intent not found on JANE: %s\n", attest.Intent)
		results = append(results, AttestationResult{
		    ElementID: 	eid,
		    Intent: 	attest.Intent,
		    Claim: 	map[string]interface{}{"error": "Intent not found on JANE"},
		    Passed: 	false,
		})
		continue
	    }
	    
	    fmt.Printf("[ATTEST] Running attestation eid=%s pid=%s intent=%s endpoint=%s\n",
		eid, pid, attest.Intent, attest.Endpoint,
	    )
	   
	    claim, err := janeRunAttestation(janeURL, eid, pid, attest.Endpoint, sid)
	    if err != nil {
		results = append(results, AttestationResult{ElementID: eid, Intent: attest.Intent, Claim: map[string]string{"error": err.Error()}, Passed: false})
		continue
	   }
	   passed := runRules(claim, attest.Rules)
	   
	   if m, ok := claim["message"].(string); ok && m == "Not Found" {
		passed = false
	   }

	   results = append(results, AttestationResult{
		ElementID: eid,
		Intent: attest.Intent,
		Claim: claim,
		Passed: passed,
	   })
	}
    }

    fmt.Printf("\n=== FINISHED EXECUTE POLICY ===\n")
    return c.JSON(http.StatusOK, map[string]interface{}{
	"results":	results,
	"count": 	len(results),
    })
}

func janeRunAttestation(janeURL, eid, pid, epn, sid string) (map[string]interface{}, error) {
    attestData := map[string]interface{}{
	"eid": 		elementID,
	"pid": 		attest.Intent,
	"epn": 		endpoint,
	"sid":		sid,
	"parameters":	map[string]interface{}{},
    }

    body, _ := json.Marshal(attestData)
    resp, err := http.Post(fmt.Sprintf("%s/attest", janeURL), "application/json", bytes.NewBuffer(body))
    if err != nil {
	return nil, fmt.Errorf("attest call failed: %v", err)
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
	return nil, fmt.Errorf("decode error: %v", err)
    }

    return result, nil
}

func debugJaneHandler(c echo.Context) error {
    port := config.ConfigData.Rest.Port
    fmt.Printf("[DEBUG] Config port: %s\n", port)

    elements, err := janeGetElementsByName("bobafet")
    if err != nil {
	return c.JSON(500, map[string]interface{}{
	    "error": "Failed to get elements",
	    "details": err.Error(),
	})
    }

    janeURL := fmt.Sprintf("http://127.0.0.1:%s", port)
    intentsResp, err := http.Get(janeBaseURL + "/intents")
    var intentsData interface{}
    if err == nil {
	defer intentsResp.Body.Close()
	json.NewDecoder(intentsResp.Body).Decode(&intentsData)
    }

    return c.JSON(200, map[string]interface{}{
	"config_port": port,
	"bobafet_elements": elements,
	"jane_intents": intentsData,
	"jane_url": janeURL,
    })
}



