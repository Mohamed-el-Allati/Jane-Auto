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
	    <form action="/execute" method="post">
		<input type="hidden" name="policy" value="%s">
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
    sid, err := createJaneSession(janeURL)
    if err != nil {
	return c.String(http.StatusInternalServerError, "Failed to create JANE session: "+err.Error())
    }
    defer closeJaneSession(janeURL, sid)

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
	   claim, err := janeRunAttestationWithSession(janeURL, eid, intentName, sid, attest.Endpoint)
	   if err != nil {
		results = append(results, AttestationResult{ElementID: eid, Intent: intentName, Claim: map[string]string{"error": err.Error()}, Passed: false})
		continue
	   }
	   passed := runRules(claim, policy)
	   results = append(results, AttestationResult{ElementID: eid, Intent: intentName, Claim: claim, Passed: passed})
	}
    }

    return c.JSON(http.StatusOK, map[string]interface{}{
	"session_id":	sid,
	"results":	results,
    })
}

func createJaneSession(janeURL string)(string, error) {
    body := []byte(`{"message": "Attestation session from handler"}`)
    resp, err := http.Post(janeURL+"/session", "application/json", bytes.NewBuffer(body))
    if err != nil {
	return "", err
    }
    defer resp.Body.Close()

    var s sessionResponse
    data, _ := ioutil.ReadAll(resp.Body)
    if err := json.Unmarshal(data, &s); err != nil {
	return "", err
    }
    return s.SID, nil
}

func closeJaneSession(janeURL, sid string) {
    req, _ := http.NewRequest("DELETE", janeURL+"/session/"+sid, nil)
    client := &http.Client{}
    client.Do(req)
}

func janeRunAttestationWithSession(janeURL, eid, intentName, sid, endpoint string) (map[string]interface{}, error) {
    payload := map[string]interface{}{
	"eid": eid,
	"epn": endpoint,
	"pid": intentName,
	"sid": sid,
    }
    b, _ := json.Marshal(payload)
    resp, err := http.Post(janeURL+"/attest", "application/json", bytes.NewBuffer(b))
    if err != nil {
	return nil, err
    }
    defer resp.Body.Close()

    var claim map[string]interface{}
    json.NewDecoder(resp.Body).Decode(&claim)
    return claim, nil
}

















