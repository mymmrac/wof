package wheel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	bolt "go.etcd.io/bbolt"
	berrors "go.etcd.io/bbolt/errors"

	"github.com/mymmrac/wof/pkg/module/id"
)

type Repository interface {
	Create(ctx context.Context, model *Model) error
	UpdateName(ctx context.Context, id id.ID, name string) error
	Get(ctx context.Context) ([]Model, error)
	GetByID(ctx context.Context, id id.ID) (*Model, bool, error)
	DeleteByID(ctx context.Context, id id.ID) error
}

type repository struct {
	db *bolt.DB
}

func NewRepository(db *bolt.DB) Repository {
	return &repository{
		db: db,
	}
}

func (r *repository) Create(_ context.Context, model *Model) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		wheels, err := tx.CreateBucketIfNotExists([]byte("wheels"))
		if err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}

		wheel, err := wheels.CreateBucket([]byte(model.ID.String()))
		if err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}

		data, err := json.Marshal(model)
		if err != nil {
			return fmt.Errorf("marshal model: %w", err)
		}

		if err = wheel.Put([]byte("model"), data); err != nil {
			return fmt.Errorf("put model: %w", err)
		}

		return nil
	})
}

func (r *repository) UpdateName(_ context.Context, id id.ID, name string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		wheels := tx.Bucket([]byte("wheels"))
		if wheels == nil {
			return fmt.Errorf("wheels bucket not found")
		}

		wheel := wheels.Bucket([]byte(id.String()))
		if wheel == nil {
			return fmt.Errorf("wheel not found")
		}

		var model Model
		if err := json.Unmarshal(wheel.Get([]byte("model")), &model); err != nil {
			return fmt.Errorf("unmarshal model: %w", err)
		}

		model.Name = name

		data, err := json.Marshal(model)
		if err != nil {
			return fmt.Errorf("marshal model: %w", err)
		}

		if err = wheel.Put([]byte("model"), data); err != nil {
			return fmt.Errorf("put model: %w", err)
		}

		return nil
	})
}

func (r *repository) Get(_ context.Context) ([]Model, error) {
	var models []Model
	if err := r.db.View(func(tx *bolt.Tx) error {
		wheels := tx.Bucket([]byte("wheels"))
		if wheels == nil {
			return nil
		}

		err := wheels.ForEachBucket(func(key []byte) error {
			wheel := wheels.Bucket(key)
			if wheel == nil {
				return nil
			}

			var model Model
			if err := json.Unmarshal(wheel.Get([]byte("model")), &model); err != nil {
				return fmt.Errorf("unmarshal model: %w", err)
			}
			models = append(models, model)

			return nil
		})
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}
	return models, nil
}

func (r *repository) GetByID(_ context.Context, id id.ID) (*Model, bool, error) {
	var model Model
	var found bool
	if err := r.db.View(func(tx *bolt.Tx) error {
		wheels := tx.Bucket([]byte("wheels"))
		if wheels == nil {
			return nil
		}

		wheel := wheels.Bucket([]byte(id.String()))
		if wheel == nil {
			return nil
		}

		if err := json.Unmarshal(wheel.Get([]byte("model")), &model); err != nil {
			return fmt.Errorf("unmarshal model: %w", err)
		}
		found = true

		return nil
	}); err != nil {
		return nil, false, err
	}
	if found {
		return &model, true, nil
	}
	return nil, false, nil
}

func (r *repository) DeleteByID(_ context.Context, id id.ID) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		wheels := tx.Bucket([]byte("wheels"))
		if wheels == nil {
			return nil
		}

		if err := wheels.DeleteBucket([]byte(id.String())); err != nil && !errors.Is(err, berrors.ErrBucketNotFound) {
			return fmt.Errorf("delete bucket: %w", err)
		}

		return nil
	})
}
