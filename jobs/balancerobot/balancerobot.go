package balancerobot

import (
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Unknwon/com"
	"github.com/boltdb/bolt"
	"github.com/zhuharev/boltutils"
	"github.com/zhuharev/jober/jobs"
	"github.com/zhuharev/jober/types"

	ini "gopkg.in/ini.v1"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

var (
	bName = []byte("balancerobot")
)

func init() {
	jobs.Register("balancerobot", New())
}

type Robot struct {
	token   string
	bot     *tgbotapi.BotAPI
	db      *bolt.DB
	adminID int64
}

func New() *Robot {
	return &Robot{}
}

func (r *Robot) Init(cfg *ini.Section) (types.Job, error) {
	time.Sleep(1 * time.Second)
	r.token = cfg.Key("token").String()
	r.adminID = cfg.Key("admin_id").MustInt64()

	err := os.MkdirAll("data/balancerobot", 0777)
	if err != nil {
		return r, err
	}

	db, err := bolt.Open("data/balancerobot/db.bolt", 0777, nil)
	if err != nil {
		return r, err
	}

	err = boltutils.CreateBucket(db, bName)
	if err != nil {
		return r, err
	}

	r.db = db

	log.Println("Start balance bot", r.token)
	r.bot, err = tgbotapi.NewBotAPIWithClient(r.token, &http.Client{})
	if err != nil {

		log.Println(err)
		return r, nil
	}

	msg := tgbotapi.NewMessage(r.adminID, "Balance bot started")
	_, err = r.bot.Send(msg)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Robot) Run() (err error) {

	log.Println("disable debug")
	r.bot.Debug = false

	log.Printf("Authorized on account %s\n", r.bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := r.bot.GetUpdatesChan(u)
	if err != nil {
		return err
	}

	for update := range updates {
		if update.Message == nil {
			continue
		}
		err = r.handler(update.Message)
		if err != nil {
			log.Println(err)
		}
	}
	return nil
}

func (r *Robot) get(id int64) (sum int64, err error) {
	bts, err := boltutils.Get(r.db, bName, i2b(id))
	if err != nil {
		if err == boltutils.ErrNotFound {
			return 0, nil
		}
		return 0, err
	}
	return b2i(bts), nil
}

func (r *Robot) set(id int64, sum int64) (err error) {
	return boltutils.Put(r.db, bName, i2b(id), i2b(sum))
}

func (r *Robot) inc(id, delta int64) error {
	sum, err := r.get(id)
	if err != nil {
		return err
	}
	sum += delta

	return r.set(id, sum)
}

func i2b(i int64) []byte {
	data := make([]byte, 8)
	binary.BigEndian.PutUint64(data, uint64(i))
	return data
}

func b2i(data []byte) int64 {
	ival := binary.BigEndian.Uint64(data)
	return int64(ival)
}

func (r *Robot) handler(m *tgbotapi.Message) error {

	// clear
	var commands = []string{"clear", "empty", "erase"}
	for _, v := range commands {
		if m.Text == "/"+v {
			return r.clear(m.Chat.ID)
		}
	}
	arr := strings.Split(m.Text, " ")
	i := com.StrTo(arr[0]).MustInt64()
	if i != 0 {
		err := r.inc(m.Chat.ID, i)
		if err != nil {
			return err
		}
	}

	sum, err := r.get(m.Chat.ID)
	if err != nil {
		return err
	}
	msg := tgbotapi.NewMessage(m.Chat.ID, fmt.Sprint(sum))
	_, err = r.bot.Send(msg)
	return err
}

func (r *Robot) clear(userID int64) error {
	return r.set(userID, 0)
}
