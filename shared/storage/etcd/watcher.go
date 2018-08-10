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
	"path/filepath"

	clientv3 "github.com/coreos/etcd/clientv3"
	generic "github.com/universonic/cmdb/shared/storage/generic"
)

type watcher struct {
	key     string
	opts    []clientv3.OpOption
	watcher clientv3.Watcher
	outChan chan generic.WatchEvent
}

func (in *watcher) Close() error {
	return in.watcher.Close()
}

func (in *watcher) Output() <-chan generic.WatchEvent {
	return in.outChan
}

func (in *watcher) watch() {
	ch := in.watcher.Watch(context.Background(), in.key, in.opts...)
	defer close(in.outChan)
	for resp := range ch {
		if err := resp.Err(); err != nil {
			in.outChan <- generic.WatchEvent{
				Type:  generic.ERROR,
				Value: []byte(err.Error()),
			}
			return
		}
		for _, event := range resp.Events {
			var t generic.WatchEventType
			if event.Type == clientv3.EventTypePut {
				if event.Kv.CreateRevision == event.Kv.ModRevision {
					t |= generic.CREATE
				} else {
					t |= generic.UPDATE
				}
			}
			if event.Type == clientv3.EventTypeDelete {
				t |= generic.DELETE
			}
			k := string(event.Kv.Key)
			ev := generic.WatchEvent{
				Type:  t,
				Kind:  filepath.Base(filepath.Dir(k)),
				Key:   filepath.Base(k),
				Value: event.Kv.Value,
			}
			in.outChan <- ev
		}
	}
}

func newWatcherFrom(w clientv3.Watcher, key string, opts ...clientv3.OpOption) generic.Watcher {
	t := &watcher{
		key:     key,
		opts:    opts,
		watcher: w,
		outChan: make(chan generic.WatchEvent, generic.DefaultWatchChanSize),
	}
	go t.watch()
	return t
}
