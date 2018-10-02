package types

import ini "gopkg.in/ini.v1"

type Job interface {
	Run() error
	Init(*ini.Section) (Job, error)
}
