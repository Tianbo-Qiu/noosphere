package db

type CredentialType string

const CredentialTypePassword CredentialType = "password"

type Traits struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}
