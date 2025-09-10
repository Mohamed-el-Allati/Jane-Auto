package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"
)

var janeURL = "http://localhost:8520"

func janeGet(path string, target interface{}) error {
    client := &http.Client{Timeout: 10*time.Second}
    resp, err := client.Get(janeURL + path)
    if err != nil {
	return fmt.Errorf("request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
	return fmt.Errorf("jane returned %s", resp.Status)
    }
    return json.NewDecoder(resp.Body).Decode(target)
}

func executePolicy(policy Policy) ([]string, error) {
    elementSet := make(map[string]struct{})

    for _, itemID := range policy.Collection.Names {
	var item Item
	if err := janeGet("/items/"+itemID, &item); err != nil {
	    return nil, fmt.Errorf("failed fetching item %s: %w", itemID, err)
	}
	if len(item.Elements) == 0 {
	    return nil, fmt.Errorf("item %s has no elements present", itemID)
	}
	for _, elemID := range item.Elements {
	    elementSet[elemID] = struct{}{}
	}
    }
    
    for _, name := range policy.Collection.Names {
	var elements []Element
	if err := janeGet("/elements/name/"+name, &elements); err != nil {
	    return nil, fmt.Errorf("failed to fetch elements by name %s: %w", name, err)
    	}
    	for _, e := range elements {
	    elementSet[e.ID] = struct{}{}
    	}
    }

    unique := make([]string, 0, len(elementSet))
    for id := range elementSet {
    	unique = append(unique, id)
    }
    return unique, nil
}
