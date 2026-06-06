package utils

import (
	"errors"
)

type ContextKey string

func AuthorizeUser(userRole string, allowedRoles ...string) error {
	for _, allowedRole := range allowedRoles {
		if userRole == allowedRole {
			return nil
		}
	}
	return errors.New("User not authorized")
}
