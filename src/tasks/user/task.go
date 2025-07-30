package user

type Task struct {
	Cond       *bool    `json:"condition"   hcl:"condition,optional" default:"true"`
	State      string   `json:"state"       hcl:"state"              validate:"in:[\"present\"]"`
	Name       string   `json:"name"        hcl:"name"               validate:"user"`
	Group      string   `json:"group"       hcl:"group,optional"     validate:"group"`
	Groups     []string `json:"groups"      hcl:"groups,optional"    validate:"group"`
	HomeDir    string   `json:"home_dir"    hcl:"home_dir,optional"`
	CreateHome bool     `json:"create_home" hcl:"create_home,optional"`
	Shell      string   `json:"shell"       hcl:"shell,optional"`
	System     bool     `json:"system"      hcl:"system,optional"`
}
