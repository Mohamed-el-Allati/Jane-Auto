package main

type Policy struct {
    Name	string			`bson:"name" json:"name"`
    Description	string			`bson:"description" json:"description"`
    Jane	string			`bson:"jane" json:"jane"`
    Collection	PolicyCollection	`bson:"collection" json:"collection"`
    Attestation map[string]AttestItem   `bson:"attestation" json:"attestation"`
}

type AttestItem struct {
    Endpoint string             `bson:"endpoint" json:"endpoint"`
    Rules    map[string]Rule    `bson:"rules" json:"rules"`
}

type Rule struct {
    RVariable string    `bson:"rvariable" json:"rvariable"`
    Parameter string    `bson:"parameter" json:"parameter"`
    Decision  string    `bson:"decision"  json:"decision"`
}

type PolicyCollection struct {
    Items	[]string `bson:"items" json:"items"`
    Tags	[]string `bson:"tags" json:"tags"`
    Names	[]string `bson:"names" json:"names"`
}

type Item struct {
    ID		string `json:"id"`
    Elements 	[]string `json:"elements"`
}

type Element struct {
    ID	string `json:"id"`
    Name string `json:"name"`
}
