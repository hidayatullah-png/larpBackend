package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hibiken/asynq"
	"latihan-fiber/app/model"
	"latihan-fiber/app/repository"
	"latihan-fiber/helper"
)

type NotificationWorker struct {
	users         repository.UserRepository
	courses       repository.CourseRepository
	notifications repository.NotificationRepository
	tx            *repository.TxManager
	mailer        helper.Mailer
	logger        *slog.Logger
}

func NewNotificationWorker(
	users repository.UserRepository,
	courses repository.CourseRepository,
	notifications repository.NotificationRepository,
	tx *repository.TxManager,
	mailer helper.Mailer,
	logger *slog.Logger,
) *NotificationWorker {
	return &NotificationWorker{
		users: users, courses: courses, notifications: notifications,
		tx: tx, mailer: mailer, logger: logger,
	}
}

func (w *NotificationWorker) HandleSend(ctx context.Context, t *asynq.Task) error {
	var job model.NotificationJob

	if job.EventType == model.EventCourseQuotaChanged {
		return w.handleQuotaChanged(ctx, job)
	}

	if err := json.Unmarshal(t.Payload(), &job); err != nil {
		return fmt.Errorf("%w: payload task rusak: %v", asynq.SkipRetry, err)
	}

	var event model.EnrollmentEvent
	// BUG DIPERBAIKI: Tambahkan pointer '&' pada event agar tidak error "non-pointer"
	if err := json.Unmarshal(job.Payload, &event); err != nil {
		return fmt.Errorf("%w: payload event %d rusak: %v", asynq.SkipRetry, job.OutboxID, err)
	}

	user, err := w.users.FindByID(ctx, event.UserID)
	if errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("%w: user %d tidak ditemukan", asynq.SkipRetry, event.UserID)
	}
	if err != nil {
		return err // database sedang bermasalah: layak di-retry
	}

	msg, err := BuildMessage(job.EventType, event, user)
	if err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}

	alreadySent := false
	err = w.tx.WithTxOnce(ctx, func(tx repository.DBTX) error {
		// BUG DIPERBAIKI: Menambahkan 'WithTx(tx)' agar claim berada dalam satu transaction
		claimed, err := w.notifications.WithTx(tx).Claim(ctx, model.Notification{
			OutboxID: job.OutboxID, UserID: user.ID, Channel: "email",
			Subject: msg.Subject, Body: msg.Body,
		})
		if err != nil {
			return err
		}
		if !claimed {
			alreadySent = true
			return nil
		}

		return w.mailer.Send(ctx, msg)
	})

	if errors.Is(err, helper.ErrPermanentDelivery) {
		// BUG DIPERBAIKI: Menggunakan format '%w: %v' agar SkipRetry bisa terbaca sistem Asynq
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}
	if err != nil {
		return err
	}

	retry, _ := asynq.GetRetryCount(ctx)
	w.logger.Info("notification_processed",
		slog.Int64("outbox_id", job.OutboxID),
		slog.String("event", job.EventType),
		slog.Int("retry", retry),
		slog.Bool("already_sent", alreadySent))
	return nil

}

func (w *NotificationWorker) handleQuotaChanged(ctx context.Context, job model.NotificationJob) error {
	var event model.QuotaChangedEvent
	if err := json.Unmarshal(job.Payload, &event); err != nil {
		return fmt.Errorf("%w: payload quota event %d rusak: %v", asynq.SkipRetry, job.OutboxID, err)
	}

	users, err := w.courses.FindEnrolledUsers(ctx, event.CourseID)
	if err != nil {
		return err
	}

	for _, user := range users {
		msg := BuildQuotaChangedMessage(event, user)
		alreadySent := false

		err := w.tx.WithTxOnce(ctx, func(tx repository.DBTX) error {
			claimed, err := w.notifications.WithTx(tx).Claim(ctx, model.Notification{
				OutboxID: job.OutboxID, UserID: user.ID, Channel: "email",
				Subject: msg.Subject, Body: msg.Body,
			})
			if err != nil {
				return err
			}
			if !claimed {
				alreadySent = true
				return nil
			}
			return w.mailer.Send(ctx, msg)
		})
		if errors.Is(err, helper.ErrPermanentDelivery) {
			return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
		}
		if err != nil {
			return err
		}

		retry, _ := asynq.GetRetryCount(ctx)
		w.logger.Info("notification_processed",
			slog.Int64("outbox_id", job.OutboxID),
			slog.Int("user_id", user.ID),
			slog.String("event", job.EventType),
			slog.Int("retry", retry),
			slog.Bool("already_sent", alreadySent))
	}
	return nil
}
