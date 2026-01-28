package main

import (
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "janeauto/config"
)

janeURL := config.ConfigData.Rest.Port

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

type returnElements struct {
    Elements []string  `json:"elements"`
    Length   int       `json:"length"`
}


func janeGetElementsByName(name string) ([]string, error){
    url := janeURL+"/elements/name/"+name
    fmt.Printf(" getting URL: %v\n", url)
    resp, err := http.Get(url)
    if err != nil {
	return nil, err
    }
    defer resp.Body.Close()

    fmt.Printf(" body is %v\n",resp.Body)

    var es returnElements 

    if err := json.NewDecoder(resp.Body).Decode(&es); err != nil {
        fmt.Printf("Decode error is %v\n",err.Error())
	return nil, err
    }
 
    fmt.Printf("Returned element is %v\n",es.Elements)
    return es.Elements, nil
}
