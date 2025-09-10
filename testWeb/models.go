package main

type Policy struct {
    Name	string			`bson:"name" json:"name"`
    Description	string			`bson:"description" json:"name"`
    Jane	string			`bson:"jane" json:"jane"`
    Collection	PolicyCollection	`bson:"collection" json:"collection"`
}


type PolicyCollection struct {
    Items	[]string `bson:"items" json:"items"`
    Tags	[]string `bson:"tags" json:"tags"`
    Names	[]string `bson:"names" json"names"`
}

type Item struct {
    ID		string `json:"id"`
    Elements 	[]string `json:"elements"`
}

type Element struct {
    ID	string `json:"id"`
    Name string `json:"name"`
}
