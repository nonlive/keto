package userdata

import "errors"

// UserData defines a user data struct
type UserData struct {
}

// New returns a new UserData struct
func New() *UserData {
	return &UserData{}
}

// Generate returns a compiled
func (u *UserData) Generate(kind string) ([]byte, error) {
	switch kind {
	case "etcd":
		return EtcdTemplate, nil
	case "master":
		return MasterTemplate, nil
	case "compute":
		return ComputeTemplate, nil
	}
	return []byte{}, errors.New("no template for this node pool kind")
}
