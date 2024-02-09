package cni

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/MikeZappa87/kni-api/pkg/apis/runtime/beta"
	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

const PodBucket = "pod"

type Store struct {
	store  *bolt.DB
}

type NetworkStorage struct {
	IP           map[string]*beta.IPConfig 
	Annotations  map[string]string 
	Extradata    map[string]string  
	Uid			 string	
	Netns_path   string	
	PodName		 string	
	PodNamespace string 
}

func New(dbName string) (*Store, error) {

	db, err := bolt.Open(dbName, 0600, nil)
	if err != nil {
		return nil, err
	}

	log.Info("boltdb connection is open")

	db.Update(func(tx *bolt.Tx) error {
		tx.CreateBucketIfNotExists([]byte(PodBucket))

		return nil
	})

	return &Store{
		store: db,
	}, nil
}

func (s *Store) Save(id string, netstore NetworkStorage) error {
	err := s.store.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(PodBucket))
		if b == nil {
			return fmt.Errorf("bucket does not exist")
		}

		if err != nil {
			return err
		}

		js, err := json.Marshal(netstore)

		if err != nil {
			return err
		}

		return b.Put([]byte(id), js)
	})

	return err
}

func (s *Store) Delete(id string) error{
	err := s.store.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(PodBucket))

		if err != nil {
			return err
		}

		return b.Delete([]byte(id))
	})

	return err
}

func (s *Store) Query(id string) (*NetworkStorage, error) {
	data := &NetworkStorage{}

	err := s.store.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(PodBucket))

		if b == nil {
			return errors.New("bucket not created")
		}

		v := b.Get([]byte(id))

		if v == nil {
			return nil
		}

		err := json.Unmarshal(v, data)

		if err != nil {
			return err
		}

		return nil
	})

	return data, err
}