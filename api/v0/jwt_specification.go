package v0

import (
	v0 "github.com/abcxyz/jvs/apis/v0"
	"github.com/golang-jwt/jwt"
)

type JVSClaims struct {
	*jwt.StandardClaims
	Justifications []*v0.Justification
}
