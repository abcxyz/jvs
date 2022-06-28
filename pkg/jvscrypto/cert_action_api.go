package jvscrypto

import (
	"context"

	jvspb "github.com/abcxyz/jvs/apis/v0"
)

// CertificateActionService lorem ipsum...
type CertificateActionService struct {
	jvspb.CertificateActionServiceServer
	Handler *RotationHandler
}

func (p *CertificateActionService) CertificateAction(ctx context.Context, request *jvspb.CertificateActionRequest) (bool, error) {
	return true, nil
}
