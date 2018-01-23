package meta

type Api struct {
	Name string				`yaml:"name"`
	Parameters []Parameter	`yaml:"parameters"`
}

type Parameter struct {
	Name string		`yaml:"name"`
	Type string 	`yaml:"type"`
	Required bool	`yaml:"required"`
	SubParameters []Parameter	`yaml:""`
}