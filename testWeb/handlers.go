package main 

import (
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

    var policies []Policy
    if err = cursor.All(ctx, &policies); err != nil {
	return c.String(http.StatusInternalServerError, "Error decoding policies")
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
    ElementID	string	`json:"element_id"`
    Intent	string	`json:"intent"`
    Claim	interface{}	`json:"claim"`
    Passed	bool	`json:"passed"`
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
    fmt.Printf("ElementIDs is %v\n",elementIDs)
    }

    var results []AttestationResult

    for _, id := range elementIDs {
	for intentName, attestItem := range policy.Attestation {
	    fmt.Printf("[attest] Processing element: %s\n", id)

	    claim, err := janeRunAttestation(id, intent)
	    if err != nil {
		fmt.Printf("[attest][ERROr] Failed to attest element %s, int %s: %v\n", id, intent, err)
		continue
	    }
	}
	fmt.Printf("[attest] Claim received for element %s, intent %s: %+v\n", id, intent, claim)

	    passed := runRules(claim, policy)
	    results = append(results, AttestationResult{
		ElementID: id,
		Intent: intent,
		Claim: claim,
		Passed: passed,
	    })
    }
    return c.JSON(http.StatusOK, results)
}

func runRules(claim map[string]interface{}, policy *Policy) bool {
    // this will load rules from yaml and evaluate claim fields, not yet done tho
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
