package db

type CredentialType string

const CredentialTypePassword CredentialType = "password"

type Traits struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
}

type PasswordConfig struct {
	HashedPassword string `json:"hashed_password"`
}

type RegistrationFlowState struct {
	Traits Traits             `json:"traits"`
	Errors RegistrationErrors `json:"errors"`
}

// RegistrationErrors holds the error message for each field, empty if the
// field is fine.
type RegistrationErrors struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (e RegistrationErrors) IsZero() bool {
	return e.Email == "" && e.Password == ""
}
