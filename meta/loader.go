package meta

type FileReader interface {
	ReadYaml(path string, v interface{}) error
}
