package models

type Policy struct {
	Name         string           `bson:"name" json:"name"`
	Description  string           `bson:"description" json:"description"`
	Jane         string           `bson:"jane" json:"jane"`
	Collection   PolicyCollection `bson:"collection" json:"collection"`
	Attestations []AttestItem     `bson:"attestations" json:"attestations"`
}

type AttestItem struct {
	Intent   string `bson:"intent" json:"intent"`
	Endpoint string `bson:"endpoint" json:"endpoint"`
	Rules    []Rule `bson:"rules" json:"rules"`
}

type Rule struct {
	Name      string `bson:"name"	  json:"name"`
	RVariable string `bson:"rvariable" json:"rvariable"`
	Parameter string `bson:"parameter" json:"parameter"`
	Decision  string `bson:"decision"  json:"decision"`
}

type PolicyCollection struct {
	Items []string `bson:"items" json:"items"`
	Tags  []string `bson:"tags" json:"tags"`
	Names []string `bson:"names" json:"names"`
}

type AttestationResult struct {
	ElementID   string                   `bson:"element_id" json:"element_id"`
	ElementName string		     `bson:"element_name" json:"element_name"`
	Intent      string                   `bson:"intent" json:"intent"`
	Claim       interface{}              `bson:"claim" json:"claim"`
	Passed      bool                     `bson:"passed" json:"passed"`
	RuleResults []map[string]interface{} `bson:"rule_results" json:"rule_results"`
	ClaimID     string                   `bson:"claim_id" json:"claim_id"`
}

type Item struct {
	ID       string   `json:"id"`
	Elements []string `json:"elements"`
}
