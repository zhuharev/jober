package jober

import (
	"time"

	"github.com/zhuharev/jober/jobs"
	"github.com/zhuharev/jober/types"
	ini "gopkg.in/ini.v1"
)

// Job represent an job

type Jober struct {
	Jobs    []types.Job
	iniFile *ini.File
}

func New(conf string) (*Jober, error) {
	iniFile, err := ini.Load(conf)
	if err != nil {
		return nil, err
	}
	jober := &Jober{
		iniFile: iniFile,
	}
	jobers := iniFile.Section("").Key("jobers").Strings(",")
	for _, v := range jobers {
		j, err := jobs.New(v, iniFile.Section(v))
		if err != nil {
			return nil, err
		}
		jober.Jobs = append(jober.Jobs, j)
	}
	return jober, nil
}

func (j *Jober) Start() error {
	for i := range j.Jobs {
		go func(i int) {
			j.Jobs[i].Run()
		}(i)
		time.Sleep(2 * time.Second)
	}
	return nil
}
