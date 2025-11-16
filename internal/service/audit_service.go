package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/hxuan190/stable_payment_gateway/internal/model"
	"github.com/hxuan190/stable_payment_gateway/internal/repository"
)

// AuditService handles audit logging for KYC operations
type AuditService interface {
	LogKYCAction(ctx context.Context, req *AuditLogRequest) error
	GetAuditLogs(ctx context.Context, submissionID uuid.UUID, limit, offset int) ([]*model.KYCAuditLog, error)
}

type auditService struct {
	repo repository.KYCRepository
}

// NewAuditService creates a new audit service
func NewAuditService(repo repository.KYCRepository) AuditService {
	return &auditService{repo: repo}
}

type AuditLogRequest struct {
	SubmissionID *uuid.UUID
	MerchantID   *uuid.UUID
	ActorType    model.ActorType
	ActorID      string
	ActorEmail   string
	Action       string
	ResourceType string
	ResourceID   *uuid.UUID
	OldStatus    string
	NewStatus    string
	Changes      *string
	IPAddress    string
	UserAgent    string
}

func (s *auditService) LogKYCAction(ctx context.Context, req *AuditLogRequest) error {
	log := &model.KYCAuditLog{
		KYCSubmissionID: req.SubmissionID,
		MerchantID:      req.MerchantID,
		ActorType:       req.ActorType,
		Action:          req.Action,
		ResourceType:    &req.ResourceType,
		ResourceID:      req.ResourceID,
		Changes:         req.Changes,
	}

	if req.ActorID != "" {
		log.ActorID = &req.ActorID
	}

	if req.ActorEmail != "" {
		log.ActorEmail = &req.ActorEmail
	}

	if req.OldStatus != "" {
		log.OldStatus = &req.OldStatus
	}

	if req.NewStatus != "" {
		log.NewStatus = &req.NewStatus
	}

	if req.IPAddress != "" {
		log.IPAddress = &req.IPAddress
	}

	if req.UserAgent != "" {
		log.UserAgent = &req.UserAgent
	}

	return s.repo.CreateAuditLog(ctx, log)
}

func (s *auditService) GetAuditLogs(ctx context.Context, submissionID uuid.UUID, limit, offset int) ([]*model.KYCAuditLog, error) {
	return s.repo.ListAuditLogsBySubmission(ctx, submissionID, limit, offset)
}
