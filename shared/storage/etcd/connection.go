// Copyright © 2018 Alfred Chou <unioverlord@gmail.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package etcd

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	clientv3 "github.com/coreos/etcd/clientv3"
	uuid "github.com/satori/go.uuid"
	generic "github.com/universonic/cmdb/shared/storage/generic"
	zap "go.uber.org/zap"
)

const (
	// DefaultRequestTimeout indicates the timeout duration of all etcd request.
	DefaultRequestTimeout = 3 * time.Second
)

type conn struct {
	prefix string // This is a workaround as Client.Watch() does not wrap keys into KV
	db     *clientv3.Client
	logger *zap.SugaredLogger
}

func (in *conn) Close() error {
	return in.db.Close()
}

func (in *conn) Create(obj generic.Object) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultRequestTimeout)
	defer cancel()
	if _, err := uuid.FromString(obj.GetGUID()); err != nil {
		obj.SetGUID(uuid.NewV4().String())
	}
	obj.SetCreationTimestamp(time.Now())
	if obj.GetName() == "" {
		obj.SetName(obj.GetGUID())
	}
	return in.txnCreate(ctx, in.keyOf(obj), obj)
}

func (in *conn) Get(cv generic.Object) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultRequestTimeout)
	defer cancel()
	return in.getKey(ctx, in.keyOf(cv), cv)
}

func (in *conn) Watch(kind string) (w generic.Watcher, err error) {
	switch kind {
	case generic.RESOURCE_MACHINE:
	case generic.RESOURCE_MACHINE_DIGEST:
	case generic.RESOURCE_MACHINE_SNAPSHOT:
	default:
		return nil, fmt.Errorf("No such kind of resource: %s", kind)
	}
	return newWatcherFrom(in.db.Watcher, strings.TrimSuffix(in.prefix+filepath.Join("/", kind), "/")+"/", clientv3.WithPrefix()), nil
}

func (in *conn) List(cv generic.ObjectList, ns ...string) (err error) {
	defer in.logger.Sync()
	ctx, cancel := context.WithTimeout(context.Background(), DefaultRequestTimeout)
	defer cancel()
	var res *clientv3.GetResponse
	if cv.HasNamespace() && len(ns) != 0 {
		res, err = in.db.Get(ctx, filepath.Join("/", cv.GetKind(), ns[0])+"/", clientv3.WithPrefix())
	} else {
		res, err = in.db.Get(ctx, filepath.Join("/", cv.GetKind())+"/", clientv3.WithPrefix())
	}
	if err != nil {
		return
	}
	length := 0
	for _, v := range res.Kvs {
		if err = cv.AppendRaw(v.Value); err != nil {
			return
		}
		length++
	}
	in.logger.Debugf("Listed %d object(s) in kind %s", length, cv.GetKind())
	return nil
}

func (in *conn) Update(obj generic.Object) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultRequestTimeout)
	defer cancel()
	return in.txnUpdate(ctx, in.keyOf(obj), func(currentValue []byte) ([]byte, error) {
		old := new(struct {
			generic.ObjectMeta `json:"metadata,omitempty"`
		})
		err := json.Unmarshal(currentValue, old)
		if err != nil {
			return nil, err
		}
		obj.SetGUID(old.GetGUID())
		obj.SetKind(old.GetKind())
		obj.SetName(old.GetName())
		obj.SetCreationTimestamp(old.GetCreationTimestamp())
		obj.SetUpdatingTimestamp(time.Now())
		return json.Marshal(obj)
	})
}

func (in *conn) Delete(obj generic.Object) error {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultRequestTimeout)
	defer cancel()
	return in.deleteKey(ctx, in.keyOf(obj))
}

func (in *conn) keyOf(obj generic.Object) string {
	if obj.HasNamespace() {
		return filepath.Join("/", obj.GetKind(), obj.GetNamespace(), obj.GetName())
	}
	return filepath.Join("/", obj.GetKind(), obj.GetName())
}

func (in *conn) txnCreate(ctx context.Context, key string, value interface{}) (err error) {
	defer in.logger.Sync()
	defer func() {
		if err != nil {
			in.logger.Errorf("Error occurred during creating data entity '%s' due to: %v", key, err)
		}
	}()
	var b []byte
	b, err = json.Marshal(value)
	if err != nil {
		return err
	}
	txn := in.db.Txn(ctx)
	var res *clientv3.TxnResponse
	res, err = txn.
		If(clientv3.Compare(clientv3.CreateRevision(key), "=", 0)).
		Then(clientv3.OpPut(key, string(b))).
		Commit()
	if err != nil {
		return err
	}
	if !res.Succeeded {
		return generic.ErrResourceAlreadyExists
	}
	in.logger.Debugf("Created key '%s': %s", key, b)
	return nil
}

func (in *conn) getKey(ctx context.Context, key string, value interface{}) (err error) {
	defer in.logger.Sync()
	defer func() {
		if err != nil {
			in.logger.Errorf("Error occurred during getting data entity '%s' due to: %v", key, err)
		}
	}()
	var r *clientv3.GetResponse
	r, err = in.db.Get(ctx, key)
	if err != nil {
		return err
	}
	if r.Count == 0 {
		return generic.ErrResourceNotFound
	}
	in.logger.Debugf("Retrieved key '%s': %s", key, r.Kvs[0].Value)
	return json.Unmarshal(r.Kvs[0].Value, value)
}

func (in *conn) txnUpdate(ctx context.Context, key string, update func(current []byte) ([]byte, error)) (err error) {
	var currentValue, updatedValue []byte
	defer in.logger.Sync()
	defer func() {
		if err != nil {
			in.logger.Errorf("Error occurred during updating data entity '%s' due to: %v", key, err)
		}
	}()
	var getResp *clientv3.GetResponse
	getResp, err = in.db.Get(ctx, key)
	if err != nil {
		return err
	}
	var modRev int64
	if len(getResp.Kvs) > 0 {
		currentValue = getResp.Kvs[0].Value
		modRev = getResp.Kvs[0].ModRevision
	}

	updatedValue, err = update(currentValue)
	if err != nil {
		return err
	}

	txn := in.db.Txn(ctx)
	var updateResp *clientv3.TxnResponse
	updateResp, err = txn.
		If(clientv3.Compare(clientv3.ModRevision(key), "=", modRev)).
		Then(clientv3.OpPut(key, string(updatedValue))).
		Commit()
	if err != nil {
		return err
	}
	if !updateResp.Succeeded {
		return fmt.Errorf("Could not update key=%q due to: Concurrent conflicting update occurred", key)
	}
	in.logger.Debugf("Updated key '%s': %s => %s", key, currentValue, updatedValue)
	return nil
}

func (in *conn) deleteKey(ctx context.Context, key string) (err error) {
	defer in.logger.Sync()
	defer func() {
		if err != nil {
			in.logger.Errorf("Error occurred during deleting data entity '%s' due to: %v", key, err)
		}
	}()
	var res *clientv3.DeleteResponse
	res, err = in.db.Delete(ctx, key)
	if err != nil {
		return err
	}
	if res.Deleted == 0 {
		return generic.ErrResourceNotFound
	}
	in.logger.Debugf("Deleted key '%s'", key)
	return nil
}
