package functions

import (
	"context"
	"encoding/csv"
	"io"
	"strings"

	"github.com/diwise/cip-functions/internal/pkg/application/functions/combinedsewageoverflow"
	"github.com/diwise/cip-functions/internal/pkg/application/functions/options"
	"github.com/diwise/cip-functions/internal/pkg/infrastructure/database"
	"github.com/diwise/messaging-golang/pkg/messaging"
	"github.com/diwise/service-chassis/pkg/infrastructure/o11y/logging"
)

type Fnct interface {
	ID() string
}

type fnctImpl struct {
	id string
	storage database.Storage
	msgCtx  messaging.MsgContext

	SewageOverflow combinedsewageoverflow.SewageOverflow
}

func (fn *fnctImpl) ID() string {
	return fn.id
}


//go:generate moq -rm -out registry_mock.go . Registry
type Registry interface {
	Find(ctx context.Context, matchers ...RegistryMatcherFunc) ([]RegistryItem, error)
}

type registry struct {
	items map[string]RegistryItem
}

type RegistryItem struct {
	FnID    string
	Name    string
	Options []options.Option
}

func NewRegistry(ctx context.Context, input io.Reader) (Registry, error) {
	logger := logging.GetFromContext(ctx)

	reg := &registry{
		items: make(map[string]RegistryItem),
	}

	r := csv.NewReader(input)
	r.Comma = ';'

	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	n := 0

	for i, row := range rows {
		if i == 0 {
			continue // skip header
		}

		item := rowToRegistryItem(row)
		reg.items[item.FnID] = item

		n++
	}

	logger.Info("loaded functions from config file", "count", n)

	return reg, nil
}

func rowToRegistryItem(row []string) RegistryItem {
	return RegistryItem{
		FnID:    row[0],
		Name:    row[1],
		Options: strToArgs(row[2]),
	}
}

func strToArgs(s string) []options.Option {
	args := make([]options.Option, 0)

	tokens := strings.Split(s, ",")
	for _, t := range tokens {
		kv := strings.Split(t, "=")
		if len(kv) == 2 {
			args = append(args, options.Option{
				Key: kv[0],
				Val: kv[1],
			})
		}
	}

	return args
}

func (r *registry) Find(ctx context.Context, matchers ...RegistryMatcherFunc) ([]RegistryItem, error) {
	fn := make([]RegistryItem, 0)

	for _, m := range matchers {
		fn = append(fn, m(r)...)
	}

	return fn, nil
}

type RegistryMatcherFunc func(r *registry) []RegistryItem

func FindByID(functionId string) RegistryMatcherFunc {
	return func(r *registry) []RegistryItem {
		fn := make([]RegistryItem, 0)

		for _, item := range r.items {
			if item.FnID == functionId {
				fn = append(fn, item)
			}
		}

		return fn
	}
}
