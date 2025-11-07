package main 

import (
    "bytes"
    "encoding/json"
    "io/ioutil"
    "context"
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

    var rawFiles []map[string]interface{}
    if err = cursor.All(ctx, &rawFiles); err != nil {
	return c.String(http.StatusInternalServerError, "Error reading the raw files: "+err.Error())
    }
    fmt.Printf("[DEBUG] Raw policies: %+v\n", rawFiles)

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

func executeHandler(c echo.Context) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    policyName := c.FormValue("policy")

    var policy Policy
    collection := client.Database("testdb").Collection("policies")
    if err := collection.FindOne(ctx, map[string]interface{}{"name": policyName}).Decode(&policy); err != nil {
	return c.String(http.StatusNotFound, "Policy not found")
    }

    elements, err := executePolicy(policy)
    if err != nil {
	return c.String(http.StatusInternalServerError, "Execution failed: "+err.Error())
    }

    accept := c.Request().Header.Get("Accept")
    if strings.Contains(accept, "json") {
	return c.JSON(http.StatusOK, map[string]interface{}{
	    "policy": policy.Name,
	    "elements": elements,
	})
    }

    var html strings.Builder
    html.WriteString(fmt.Sprintf("<h1>Executed Policy: %s</h1>", policy.Name))
    html.WriteString("<ul>")
    for _, e := range elements {
	html.WriteString(fmt.Sprintf("<li>%s</li>", e))
    }
    html.WriteString("</ul>")
    html.WriteString(`<p><a href="/policies">Back to policies</a></p>`)

    return c.HTML(http.StatusOK, html.String())
}

type AttestationResult struct {
    ElementID	string		`bson:"element_id" json:"element_id"`
    Intent	string		`bson: "intent" json:"intent"`
    Claim	interface{}	`bson: "claim" json:"claim"`
    Passed	bool		`bson: "passed" json:"passed"`
}

func attestPolicyHandler(c echo.Context) error {
    policyName := c.Param("policyName")
    fmt.Printf("[attest] Start attesting policy: %s\n", policyName)

    policy, err := dbGetPolicyByName(policyName)
    if err != nil {
	fmt.Printf("[attest][ERROR] unable to fetc policy: %v\n", err)
	return c.String(http.StatusNotFound, "Policy not Found")
    }
    fmt.Printf("[attest] LoAded policy: %+v\n", policy)

    var elementIDs []string

    elementIDs = append(elementIDs, policy.Collection.Items...)
    for _, name := range policy.Collection.Names {
	fmt.Printf("[attest] Getting name %v\n",name)
	ids, err := janeGetElementsByName(name)
        fmt.Printf("[attest] Returned ids is %v\n",ids)
	if err != nil {
            fmt.Printf("[attest][ERROR]Error being returned is %v whcih means the name wasn't found\n",err.Error())
	} else {
	    elementIDs = append(elementIDs, ids...)
	    fmt.Printf("[attest] IDs returned for %v: %v\n", name, ids)
        }
    }

    elementIDs = unique(elementIDs)
    fmt.Printf("[attest] ElementIDs is %v\n",elementIDs)

    var results []AttestationResult

    for _, id := range elementIDs {
	for intentName := range policy.Attestation {
	    fmt.Printf("[attest] Processing element: %s, intent: %s\n", id, intentName)

	    claim, err := janeRunAttestation(id, intentName)
	    if err != nil {
		fmt.Printf("[attest][ERROr] Failed to attest element %s, intent %s: %v\n", id, intentName, err)
		results = append(results, AttestationResult{
		    ElementID: 	id,
		    Intent:	intentName,
		    Claim:	map[string]interface{}{"message": err.Error()},
		    Passed:	false,
		})
		continue
	    }
	    fmt.Printf("[attest] Claim received for element %s, intent %s: %+v\n", id, intentName, claim)

	    passed := true
	    if m, ok := claim["message"].(string); ok && m == "Not Found" {
		passed = false
	    }

	    results = append(results, AttestationResult{
		ElementID: id,
		Intent: intentName,
		Claim: claim,
		Passed: passed,
	    })
	}
    }
    fmt.Printf("[attest] Completed attesting for policy: %s\n", policyName)
    return c.JSON(http.StatusOK, results)
}

func runRules(claim map[string]interface{}, policy *Policy) bool {
    if msg, ok := claim["message"]; ok && msg == "Not Found" {
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

type sessionResponse struct {
    SID		string `bson: "sid" json:"sid"`
    Status	string `bson: "status" json:"status"`
    Error	string `bson: "error" json:"error"`
}

func executePolicyHandler(c echo.Context) error {
    policyName := c.Param("policyName")

    policy, err := dbGetPolicyByName(policyName)
    if err != nil {
	return c.String(http.StatusNotFound, "Policy not found")
    }

    janeURL := policy.Jane
   
    resp, err := http.Get(janeURL + "/intents")
    if err != nil {
	return c.String(http.StatusInternalServerError, "Failed to fetch intents: "+err.Error())
    }
    defer resp.Body.Close()
    var intentData struct {
	Intents []string `bson:"intents" json:"intents"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&intentData); err != nil {
	return c.String(http.StatusInternalServerError, "Failed to decode intents: "+err.Error())
    }
    validIntents := make(map[string]struct{})
    for _, i := range intentData.Intents {
	validIntents[i] = struct{}{}
    }

    var elementIDs []string
    elementIDs = append(elementIDs, policy.Collection.Items...)
    for _, name := range policy.Collection.Names {
	ids, _ := janeGetElementsByName(name)
	elementIDs = append(elementIDs, ids...)
    }
    elementIDs = unique(elementIDs)

    var results []AttestationResult
    for _, eid := range elementIDs {
	for intentName, attest := range policy.Attestation {
	    if _, ok := validIntents[intentName]; !ok {
		results = append(results, AttestationResult{
		    ElementID: eid,
		    Intent: intentName,
		    Claim: map[string]string{"error": "Intent not found on JANE"},
		    Passed: false,
		})
		continue
	    }
	    
	    claim, err := janeSimple(janeURL, eid, intentName, attest.Endpoint)
	    if err != nil {
		results = append(results, AttestationResult{ElementID: eid, Intent: intentName, Claim: map[string]string{"error": err.Error()}, Passed: false})
		continue
	   }
	   passed := runRules(claim, policy)
	   results = append(results, AttestationResult{ElementID: eid, Intent: intentName, Claim: claim, Passed: passed})
	}
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
	"results":	results,
    })
}

func janeRunAttestation(policyName string, policy Policy, janeURL string) []Result {
    var results []Result

    for _, elementName := range policy.Collection.Names {
	fmt.Printf("[DEBUG] Processing element: %s\n", elementName)

	elementURL := fmt.Sprintf("%s/element/%s", janeURL, elementName)
	resp, err := http.Get(ElementURL)
	if err != nil {
	    fmt.Printf("[ERROR] Failed to get element %s: %v\n", elementName, err)
	    results = append(results, Result{
	    	ElementID: "",
		Intent:	"",
		Claim:	"",
		Error:	fmt.Sprintf("Failed to get element: %v", err),
		Passed: false,
	    })
	    continue
	}
	defer resp.Body.Close()

	var elementResp struct {
	    ItemID string `json:"itemid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&elementResp); err != nil {
	    fmt.Printf("[ERROR] Could not parse element response: %v\n", err)
	    results = append(results, Result {
		ElementID: "",
		Intent: "",
		Claim: "",
		Error: "Could not decode element response",
		Passed: false,
	    })
	    continue
	}

	if elementResp.ItemID == "" {
	    fmt.Printf("[DEBUG] Element %s not found on JANE - Creating... \n", elementName)
	    elementData := map[string]interface{}{
		"name": elementName,
		"tags": policy.Collection.Tags,
	    }
	    body, _ := json.Marshal(elementData)
	    createResp, err := http.Post(fmt.Sprintf("%s/element", janeURL), "application/json", bytes.NewBuffer(body))
	    if err != nil {
	   	fmt.Printf("[ERROR] Failed to create element %s: %v\n", elementName, err)
	    	results = append(results, Result{
	   	    ElementID: "", 
	 	    Intent: "",
		    Error:	fmt.Sprintf("Element creation failed: %v", err),
	 	    Passed: false,
	        })
		continue
	    }
	    defer createResp.Body.Close()
	    if err := json.NewDecoder(createResp.Body).Decode(&elementResp); err != nil {
		fmt.Printf("[ERROR] Could not decode new element creation response: %v\n", err)
		continue
	    }
	}
	
	elementID := elementResp.ItemID
	fmt.Printf("[DEBUG] Attesting element %s (%s)\n", elementName, elementID)

	for pid := range policy.Attestation {
	    intent := policy.Attestation[pid]
	    fmt.Printf("[DEBUG] Attesting element %s with intent %s\n", elementName, pid)

	    attestData := map[string]interface{}{
		"eid": elementID,
		"pid": pid,
		"epn": intent.Endpoint,
	    }

	    body, _ := json.Marshal(attestData)
	    attestURL := fmt.Sprintf("%s/attest", janeURL)
	    resp, err := http.Post(attestURL, "application/json", bytes,NewBuffer(body))

	    if err != nil {
		fmt.Printf("[ERROR] Attest call failed for %s: %v\n", pid, err)
		results = append(results, Result{
		    ElementID: elementID,
		    Intent: pid,
		    Error: fmt.Sprintf("Attest failed: %v", err),
		    Passed: false,
		})
		continue
	    }
	    defer resp.Body.Close()

	    var attestResp struct {
		ItemID string `json:"itemid"`
		Error string `json:"error"`
	    }
	    if err := json.NewDecoder(resp.Body).Decode(&attestResp); err != nil {
		results = append(results, Result{
		    ElementID: elementID,
		    Intent: pid,
		    Error: fmt.Sprintf("Invalid response: %v", err),
		    Passed: false,
		})
		continue
	    }
	    
	    passed := attestResp.Error == ""
	    results = append(results, Result{
		ElementID: elementID,
		Intent: pid,
		Claim: 	attestResp.ItemID,
		Error: 	attestResp.Error,
		Passed:	passed,
	    })
	}
    }

    return results
}





