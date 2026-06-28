package item

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/mymmrac/wof/pkg/module/id"
)

type Repository interface {
	Create(ctx context.Context, model *Model) error
	UpdateInfo(ctx context.Context, id id.ID, name string) error
	UpdateImage(ctx context.Context, id id.ID, image []byte) error
	UpdateRating(ctx context.Context, id id.ID, rating int) error
	UpdateRejected(ctx context.Context, id id.ID, rejected bool) error
	GetByID(ctx context.Context, id id.ID) (*Model, bool, error)
	GetImageByID(ctx context.Context, id id.ID) ([]byte, bool, error)
	GetByWheelID(ctx context.Context, wheelID id.ID) ([]Model, error)
	CountByWheelID(ctx context.Context, wheelID id.ID) (int, error)
	DeleteByID(ctx context.Context, id id.ID) error
	UpdateOrder(ctx context.Context, ids []id.ID) error
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
		wheels := tx.Bucket([]byte("wheels"))
		if wheels == nil {
			return fmt.Errorf("wheels bucket not found")
		}

		wheel := wheels.Bucket([]byte(model.WheelID.String()))
		if wheel == nil {
			return fmt.Errorf("wheel not found")
		}

		items, err := wheel.CreateBucketIfNotExists([]byte("items"))
		if err != nil {
			return fmt.Errorf("create items bucket: %w", err)
		}

		item, err := items.CreateBucket([]byte(model.ID.String()))
		if err != nil {
			return fmt.Errorf("create item bucket: %w", err)
		}

		data, err := json.Marshal(model)
		if err != nil {
			return fmt.Errorf("marshal model: %w", err)
		}

		if err = item.Put([]byte("model"), data); err != nil {
			return fmt.Errorf("put model: %w", err)
		}

		return nil
	})
}

func (r *repository) UpdateInfo(_ context.Context, id id.ID, name string) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		itemBucket, err := r.getItemBucket(tx, id)
		if err != nil {
			return err
		}

		var model Model
		if err = json.Unmarshal(itemBucket.Get([]byte("model")), &model); err != nil {
			return fmt.Errorf("unmarshal model: %w", err)
		}

		model.Name = name
		model.UpdatedAt = time.Now()

		data, err := json.Marshal(model)
		if err != nil {
			return fmt.Errorf("marshal model: %w", err)
		}

		if err = itemBucket.Put([]byte("model"), data); err != nil {
			return fmt.Errorf("put model: %w", err)
		}

		return nil
	})
}

func (r *repository) UpdateImage(_ context.Context, id id.ID, image []byte) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		itemBucket, err := r.getItemBucket(tx, id)
		if err != nil {
			return err
		}

		if image == nil {
			if err = itemBucket.Delete([]byte("image")); err != nil {
				return fmt.Errorf("delete image: %w", err)
			}
		} else {
			if err = itemBucket.Put([]byte("image"), image); err != nil {
				return fmt.Errorf("put image: %w", err)
			}
		}

		var model Model
		if err = json.Unmarshal(itemBucket.Get([]byte("model")), &model); err != nil {
			return fmt.Errorf("unmarshal model: %w", err)
		}

		model.UpdatedAt = time.Now()

		data, err := json.Marshal(model)
		if err != nil {
			return fmt.Errorf("marshal model: %w", err)
		}

		if err = itemBucket.Put([]byte("model"), data); err != nil {
			return fmt.Errorf("put model: %w", err)
		}

		return nil
	})
}

func (r *repository) UpdateRating(_ context.Context, id id.ID, rating int) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		itemBucket, err := r.getItemBucket(tx, id)
		if err != nil {
			return err
		}

		var model Model
		if err = json.Unmarshal(itemBucket.Get([]byte("model")), &model); err != nil {
			return fmt.Errorf("unmarshal model: %w", err)
		}

		model.Rating = rating
		model.UpdatedAt = time.Now()

		data, err := json.Marshal(model)
		if err != nil {
			return fmt.Errorf("marshal model: %w", err)
		}

		if err = itemBucket.Put([]byte("model"), data); err != nil {
			return fmt.Errorf("put model: %w", err)
		}

		return nil
	})
}

func (r *repository) UpdateRejected(_ context.Context, id id.ID, rejected bool) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		itemBucket, err := r.getItemBucket(tx, id)
		if err != nil {
			return err
		}

		var model Model
		if err = json.Unmarshal(itemBucket.Get([]byte("model")), &model); err != nil {
			return fmt.Errorf("unmarshal model: %w", err)
		}

		model.Rejected = rejected
		model.UpdatedAt = time.Now()

		data, err := json.Marshal(model)
		if err != nil {
			return fmt.Errorf("marshal model: %w", err)
		}

		if err = itemBucket.Put([]byte("model"), data); err != nil {
			return fmt.Errorf("put model: %w", err)
		}

		return nil
	})
}

func (r *repository) GetByID(_ context.Context, id id.ID) (*Model, bool, error) {
	var model Model
	var found bool
	if err := r.db.View(func(tx *bolt.Tx) error {
		itemBucket, err := r.getItemBucket(tx, id)
		if err != nil {
			return nil
		}

		data := itemBucket.Get([]byte("model"))
		if data == nil {
			return nil
		}

		if err = json.Unmarshal(data, &model); err != nil {
			return fmt.Errorf("unmarshal model: %w", err)
		}
		found = true

		return nil
	}); err != nil {
		return nil, false, err
	}

	if !found {
		return nil, false, nil
	}

	return &model, true, nil
}

func (r *repository) GetImageByID(_ context.Context, id id.ID) ([]byte, bool, error) {
	var image []byte
	var found bool
	if err := r.db.View(func(tx *bolt.Tx) error {
		itemBucket, err := r.getItemBucket(tx, id)
		if err != nil {
			return nil
		}

		data := itemBucket.Get([]byte("image"))
		if data == nil {
			return nil
		}

		image = make([]byte, len(data))
		copy(image, data)
		found = true

		return nil
	}); err != nil {
		return nil, false, err
	}
	return image, found, nil
}

func (r *repository) GetByWheelID(_ context.Context, wheelID id.ID) ([]Model, error) {
	var models []Model
	if err := r.db.View(func(tx *bolt.Tx) error {
		wheels := tx.Bucket([]byte("wheels"))
		if wheels == nil {
			return nil
		}

		wheel := wheels.Bucket([]byte(wheelID.String()))
		if wheel == nil {
			return nil
		}

		items := wheel.Bucket([]byte("items"))
		if items == nil {
			return nil
		}

		return items.ForEachBucket(func(k []byte) error {
			itemBucket := items.Bucket(k)
			data := itemBucket.Get([]byte("model"))
			if data == nil {
				return nil
			}

			var model Model
			if err := json.Unmarshal(data, &model); err != nil {
				return fmt.Errorf("unmarshal model: %w", err)
			}
			models = append(models, model)
			return nil
		})
	}); err != nil {
		return nil, err
	}
	return models, nil
}

func (r *repository) CountByWheelID(_ context.Context, wheelID id.ID) (int, error) {
	var count int
	if err := r.db.View(func(tx *bolt.Tx) error {
		wheels := tx.Bucket([]byte("wheels"))
		if wheels == nil {
			return nil
		}

		wheel := wheels.Bucket([]byte(wheelID.String()))
		if wheel == nil {
			return nil
		}

		items := wheel.Bucket([]byte("items"))
		if items == nil {
			return nil
		}

		count = items.Stats().KeyN
		return nil
	}); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *repository) DeleteByID(_ context.Context, id id.ID) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		wheels := tx.Bucket([]byte("wheels"))
		if wheels == nil {
			return nil
		}

		return wheels.ForEachBucket(func(k []byte) error {
			wheel := wheels.Bucket(k)
			items := wheel.Bucket([]byte("items"))
			if items == nil {
				return nil
			}

			if items.Bucket([]byte(id.String())) != nil {
				return items.DeleteBucket([]byte(id.String()))
			}

			return nil
		})
	})
}

func (r *repository) UpdateOrder(_ context.Context, ids []id.ID) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		for i, itemID := range ids {
			itemBucket, err := r.getItemBucket(tx, itemID)
			if err != nil {
				return fmt.Errorf("get item bucket %s: %w", itemID, err)
			}

			var model Model
			if err = json.Unmarshal(itemBucket.Get([]byte("model")), &model); err != nil {
				return fmt.Errorf("unmarshal model: %w", err)
			}

			model.Order = i
			model.UpdatedAt = time.Now()

			data, err := json.Marshal(model)
			if err != nil {
				return fmt.Errorf("marshal model: %w", err)
			}

			if err = itemBucket.Put([]byte("model"), data); err != nil {
				return fmt.Errorf("put model: %w", err)
			}
		}
		return nil
	})
}

func (r *repository) getItemBucket(tx *bolt.Tx, id id.ID) (*bolt.Bucket, error) {
	wheels := tx.Bucket([]byte("wheels"))
	if wheels == nil {
		return nil, fmt.Errorf("wheels bucket not found")
	}

	var foundBucket *bolt.Bucket
	err := wheels.ForEachBucket(func(k []byte) error {
		wheel := wheels.Bucket(k)
		items := wheel.Bucket([]byte("items"))
		if items == nil {
			return nil
		}

		item := items.Bucket([]byte(id.String()))
		if item != nil {
			foundBucket = item
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if foundBucket == nil {
		return nil, fmt.Errorf("item not found")
	}

	return foundBucket, nil
}
