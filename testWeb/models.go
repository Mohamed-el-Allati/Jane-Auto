package main

type Policy struct {
    Name	string			`bson:"name" json:"name"`
    Description	string			`bson:"description" json:"description"`
    Jane	string			`bson:"jane" json:"jane"`
    Collection	PolicyCollection	`bson:"collection" json:"collection"`
    Attestations []AttestItem		`bson:"attestations" json:"attestations"`
}

type AttestItem struct {
    Intent	string	`bson:"intent" json:"intent"`
    Endpoint	string	`bson:"endpoint" json:"endpoint"`
    Rules    	[]Rule  `bson:"rules" json:"rules"`
}

type Rule struct {
    Name      string	`bson:"name"	  json:"name"`
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
