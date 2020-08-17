package user

import (
	api "kubedb.dev/apimachinery/apis/kubedb/v1alpha1"

	"golang.org/x/crypto/bcrypt"
)

// returns true if the user exists.
// otherwise false.
func HasUser(userList map[string]api.ElasticsearchUserSpec, username api.ElasticsearchInternalUser) bool {
	if _, exist := userList[string(username)]; exist {
		return true
	}
	return false
}

// Set user if missing
func SetMissingUser(userList map[string]api.ElasticsearchUserSpec, username api.ElasticsearchInternalUser, userSpec api.ElasticsearchUserSpec) {
	if HasUser(userList, username) {
		return
	}

	userList[string(username)] = userSpec
}

func SetPasswordHashForUser(userList map[string]api.ElasticsearchUserSpec, username string, password string) error {
	var userSpec api.ElasticsearchUserSpec
	if value, exist := userList[username]; exist {
		userSpec = value
	}

	hash, err := generatePasswordHash(password)
	if err != nil {
		return err
	}

	userSpec.Hash = hash
	userList[username] = userSpec
	return nil
}

func generatePasswordHash(password string) (string, error) {
	pHash, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", err
	}
	return string(pHash), nil
}
