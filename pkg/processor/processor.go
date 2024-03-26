package processor

import (
	"context"

	"github.com/ottogroup/penelope/pkg/http/auth/model"
	"github.com/ottogroup/penelope/pkg/repository"
)

// Arguments for a Processor
type Argument[R any] struct {
	Request   R
	Principal *model.Principal
}

// Operations define operations for processors
type Operation[T, R any] interface {
	Process(context.Context, *Argument[T]) (R, error)
}

func isBackupStatusTransitionValid(current repository.BackupStatus, new repository.BackupStatus) (isValid bool) {
	switch current {
	case repository.NotStarted:
		switch new {
		case repository.Prepared, repository.Finished, repository.Paused, repository.ToDelete, repository.BackupDeleted:
			isValid = true
		}
	case repository.Prepared:
		switch new {
		case repository.NotStarted, repository.Finished, repository.Paused, repository.ToDelete, repository.BackupDeleted:
			isValid = true
		}
	case repository.Finished:
		switch new {
		case repository.NotStarted, repository.Paused, repository.ToDelete, repository.BackupDeleted:
			isValid = true
		}
	case repository.Paused:
		switch new {
		case repository.NotStarted, repository.ToDelete, repository.BackupDeleted:
			isValid = true
		}
	case repository.ToDelete:
		switch new {
		case repository.NotStarted, repository.BackupDeleted:
			isValid = true
		}
	case repository.BackupDeleted:
		switch new {
		case repository.NotStarted:
			isValid = true
		}
	}
	return isValid
}
