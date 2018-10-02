package main

import (
	"log"
	"sync"

	"github.com/zhuharev/jober"
	//_ "github.com/zhuharev/jober/jobs/balancerobot"
	//_ "github.com/zhuharev/jober/jobs/geolbot"
	_ "github.com/zhuharev/jober/jobs/readovka"
	//_ "github.com/zhuharev/jober/jobs/ug"
	//_ "github.com/zhuharev/jober/jobs/vkim"
)

func main() {
	j, err := jober.New("conf/app.ini")
	if err != nil {
		log.Fatalln(err)
	}
	err = j.Start()
	if err != nil {
		log.Fatalln(err)
	}

	var wg sync.WaitGroup
	wg.Add(1)

	wg.Wait()
}
