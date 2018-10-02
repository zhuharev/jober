package jobs

import (
	"fmt"
	"sync"

	"github.com/zhuharev/jober/types"
	ini "gopkg.in/ini.v1"
)

var jobs = map[string]types.Job{}

var mu sync.Mutex

func Register(name string, job types.Job) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := jobs[name]; ok {
		panic(name)
	}
	jobs[name] = job
}

func New(name string, cfg *ini.Section) (types.Job, error) {
	mu.Lock()
	defer mu.Unlock()
	if _, has := jobs[name]; !has {
		return nil, fmt.Errorf("unknown job %s", name)
	}
	job, err := jobs[name].Init(cfg)
	jobs[name] = job
	return jobs[name], err
}
