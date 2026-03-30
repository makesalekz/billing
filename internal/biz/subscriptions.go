package biz

import (
	"context"
	"strings"
	"time"

	v1 "gitlab.calendaria.team/services/finance/billing/api/billing/v1"
	"gitlab.calendaria.team/services/finance/billing/ent"
	"gitlab.calendaria.team/services/finance/billing/internal/data"
	utils_v1 "gitlab.calendaria.team/services/utils/api/utils/v1"
)

type SubscriptionList struct {
	Subscriptions []*ent.Subscriptions
	PaginateReply *utils_v1.PaginateReply
}

type SubscriptionsUseCase struct {
	subscriptionRepo data.SubscriptionsRepo
	invoiceRepo      data.InvoicesRepo
	productRepo      data.ProductRepo
}

func NewSubscriptionUsecase(
	subscriptionRepo data.SubscriptionsRepo,
	invoiceRepo data.InvoicesRepo,
	productRepo data.ProductRepo,
) *SubscriptionsUseCase {
	return &SubscriptionsUseCase{
		subscriptionRepo: subscriptionRepo,
		invoiceRepo:      invoiceRepo,
		productRepo:      productRepo,
	}
}

func (uc *SubscriptionsUseCase) CreateSubscription(
	ctx context.Context, actorID, tenantID int64, appID string, subscriptionDto data.SubscriptionDto,
) (*ent.Subscriptions, error) {
	subscription, err := uc.subscriptionRepo.CreateSubscription(ctx, actorID, tenantID, appID, subscriptionDto)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to create subscription, err %s", err.Error())
	}

	return subscription, nil
}

func (uc *SubscriptionsUseCase) GetSubscription(
	ctx context.Context, actorID, tenantID int64, appID string, subscriptionID int64, withInvoices bool,
) (*ent.Subscriptions, error) {
	subscription, err := uc.subscriptionRepo.GetSubscription(ctx, actorID, tenantID, appID, subscriptionID,
		withInvoices)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to get subscription, err %s", err.Error())
	}

	return subscription, nil
}

func (uc *SubscriptionsUseCase) DeleteSubscription(ctx context.Context, actorID, subscriptionID int64) error {
	err := uc.subscriptionRepo.DeleteSubscription(ctx, actorID, subscriptionID)
	if err != nil {
		return v1.ErrorDatabaseQuery("failed to delete subscription, err %s", err.Error())
	}

	return nil
}

func (uc *SubscriptionsUseCase) ListSubscriptions(
	ctx context.Context, actorID int64, withInvoices bool, paginate *utils_v1.PaginateRequest,
) (*SubscriptionList, error) {
	subscriptions, err := uc.subscriptionRepo.ListSubscriptions(ctx, actorID, withInvoices, paginate)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to list subscriptions, err %s", err.Error())
	}

	total, err := uc.subscriptionRepo.CountSubscriptions(ctx, actorID)
	if err != nil {
		return nil, v1.ErrorDatabaseQuery("failed to list subscriptions ,err %s", err.Error())
	}

	paginateReply := &utils_v1.PaginateReply{
		Total: &total,
	}

	if len(subscriptions) > 0 {
		paginateReply.FromId = &subscriptions[len(subscriptions)-1].ID
	}

	return &SubscriptionList{
		Subscriptions: subscriptions,
		PaginateReply: paginateReply,
	}, nil
}

func (uc *SubscriptionsUseCase) GetSubscriptionStatus(
	ctx context.Context, userID, tenantID int64, appID string,
) (*v1.SubscriptionStatusReply, error) {
	subs, err := uc.subscriptionRepo.GetSubscriptionsByUser(ctx, userID, tenantID, appID)
	if err != nil {
		return nil, err
	}

	if len(subs) == 0 {
		return &v1.SubscriptionStatusReply{
			HasSubscription: false,
			Plan:            "starter",
			Status:          "expired",
		}, nil
	}

	// Find the subscription with the latest paid invoice
	var bestInvoice *ent.Invoice
	var bestSub *ent.Subscriptions
	var bestProduct *ent.Product

	for _, sub := range subs {
		inv, err := uc.invoiceRepo.GetLatestPaidInvoiceBySubscription(ctx, sub.ID)
		if err != nil || inv == nil {
			continue
		}
		if bestInvoice == nil || (inv.PaidTill != nil && bestInvoice.PaidTill != nil && inv.PaidTill.After(*bestInvoice.PaidTill)) {
			bestInvoice = inv
			bestSub = sub
		}
	}

	if bestInvoice == nil || bestSub == nil {
		return &v1.SubscriptionStatusReply{
			HasSubscription: false,
			Plan:            "starter",
			Status:          "expired",
		}, nil
	}

	bestProduct, _ = uc.productRepo.GetProduct(ctx, bestSub.ProductID)

	reply := &v1.SubscriptionStatusReply{
		HasSubscription: true,
		ProductId:       bestSub.ProductID,
		SubscriptionId:  bestSub.ID,
	}

	if bestProduct != nil {
		reply.ProductName = bestProduct.Name
		reply.Plan = extractPlanSlug(bestProduct.Name)
	}

	now := time.Now()
	if bestInvoice.PaidTill != nil {
		reply.PaidTill = bestInvoice.PaidTill.Format(time.RFC3339)

		if bestInvoice.PaidTill.After(now) {
			if bestInvoice.IsRevoked {
				reply.Status = "cancelled"
				reply.AutoRenew = false
				if bestInvoice.RevokedAt != nil {
					reply.CancelledAt = bestInvoice.RevokedAt.Format(time.RFC3339)
				}
			} else {
				reply.Status = "active"
				reply.AutoRenew = true
			}
		} else {
			reply.Status = "expired"
			reply.AutoRenew = false
		}
	}

	return reply, nil
}

// extractPlanSlug extracts "pro" from "Qalai Pro Monthly"
func extractPlanSlug(name string) string {
	words := strings.Fields(strings.ToLower(name))
	if len(words) < 2 {
		return "starter"
	}
	last := words[len(words)-1]
	if last == "monthly" || last == "yearly" {
		words = words[1 : len(words)-1]
	} else {
		words = words[1:]
	}
	return strings.Join(words, "-")
}

func ReplySubscription(sub *ent.Subscriptions) *v1.Subscription {
	return &v1.Subscription{
		Id:        sub.ID,
		UserId:    sub.UserID,
		TenantId:  sub.TenantID,
		AppId:     sub.AppID,
		ProductId: sub.ProductID,
	}
}

func ReplySubscriptions(subs []*ent.Subscriptions) []*v1.Subscription {
	reply := make([]*v1.Subscription, len(subs))

	for i, sub := range subs {
		reply[i] = ReplySubscription(sub)
	}

	return reply
}
