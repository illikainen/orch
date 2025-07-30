package group

type Task struct {
	Cond   *bool    `json:"condition" hcl:"condition,optional" default:"true"`
	State  string   `json:"state"     hcl:"state"              validate:"in:[\"present\"]"`
	Name   string   `json:"name"      hcl:"name"               validate:"group"`
	Users  []string `json:"users"     hcl:"users,optional"     validate:"user"`
	System bool     `json:"system"    hcl:"system,optional"`
}
