package ug

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"time"

	"github.com/zhuharev/jober/jobs"
	"github.com/zhuharev/jober/types"
	"github.com/zhuharev/vkutil"
	ini "gopkg.in/ini.v1"
	"gopkg.in/telegram-bot-api.v4"

	"encoding/json"
	"net/http"
	"net/url"

	"github.com/ahmdrz/goinsta"
	dry "github.com/ungerik/go-dry"
)

func init() {
	jobs.Register("ug", New())

	log.SetFlags(log.Llongfile | log.LstdFlags)
}

type Server struct {
	m   []int
	tpl *template.Template
}

func (s *Server) Init(conf *ini.Section) (types.Job, error) {
	bts := MustAsset("index.tmpl")
	tpl, err := template.New("alah").Parse(string(bts))
	if err != nil {
		return nil, err
	}
	s.tpl = tpl
	return s, nil
}

func (s *Server) job() error {
	for i, v := range socs {
		cnt, err := GetFollowers(v)
		if err != nil {
			log.Println(v.SocType, err)
			continue
		}
		if cnt != 0 {
			s.m[i] = cnt
		}
	}

	f, err := os.OpenFile("/home/god/sites/ug.dev.zhuharev.ru/public/index.html", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0777)
	if err != nil {
		log.Println(err)
		return err
	}
	d := s.m
	err = s.tpl.Execute(f, map[string]interface{}{"socs": d})
	if err != nil {
		log.Println(err)
		return err
	}
	f.Close()
	err = dry.FileSetJSON("/home/god/sites/ug.dev.zhuharev.ru/public/nums.json", s.m)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (s *Server) Run() (err error) {

	err = s.job()
	if err != nil {
		return err
	}
	t := time.NewTicker(6 * time.Hour)
	for {
		select {
		case <-t.C:
			err = s.job()
			if err != nil {
				log.Println(err)
			}
		}
	}

	return nil
}

func New() *Server {
	return &Server{m: make([]int, len(socs))}
}

/////////////////////

type Soc struct {
	ID      interface{}
	SocType SocType
}

func NewSoc(id interface{}, typ SocType) Soc {
	return Soc{
		id, typ,
	}
}

type SocType int

const (
	Vk SocType = iota + 1
	T
	Insta
)

func GetFollowers(s Soc) (int, error) {
	switch s.SocType {
	case Vk:
		return vkFollowers(s.ID.(int))
	case T:
		return tgFollowers(s.ID)
	case Insta:
		return instagramFollowers(s.ID.(string))
	}
	return 0, fmt.Errorf("Unknown soc")
}

///////////////////

func instagramFollowers(id string) (int, error) {
	return 0, nil
	u, _ := url.Parse(goinsta.GOINSTA_API_URL)

	bts, err := dry.FileGetBytes("conf/cookies")
	if err != nil {
		return 0, err
	}

	var c []*http.Cookie
	err = json.Unmarshal(bts, &c)
	if err != nil {
		return 0, err
	}

	insta := goinsta.New("zhuharev", "I1m1e2e3p5o8")
	insta.IsLoggedIn = true
	err = insta.SetCookies(u, c)
	if err != nil {
		return 0, err
	}

	//if err := insta.Login(); err != nil {
	//	panic(err)
	//}

	defer func() {

		cookies := insta.GetSessions(u)

		dry.FileSetJSON("cookies", cookies)
	}()

	ur, err := insta.GetUserByUsername(id)
	if err != nil {
		return 0, err
	}
	return ur.User.FollowerCount, nil

}

/////////////////////

var (
	instaID    = "uytnoe_gnezdo"
	instaMSKID = "uytnoe_gnezdo_msk"

	vkID    = 57466174
	vkMSKID = 95396194

	tChatID    = -1001106968756
	tID        = "ughome"
	tMSKID     = "ugnezdishko"
	tChatMSKID = -1001133425345

	socs = []Soc{
		NewSoc(instaID, Insta),
		NewSoc(instaMSKID, Insta),
		NewSoc(vkID, Vk),
		NewSoc(vkMSKID, Vk),
		NewSoc(tChatID, T),
		NewSoc(tID, T),
		NewSoc(tMSKID, T),
		NewSoc(tChatMSKID, T),
	}
)

func vkFollowers(id int) (int, error) {
	u := vkutil.New()
	u.VkApi.AccessToken = "a43640f024bdf712397bf689fae3f080bf41d247c0a0d1a716ede557cb62c1f24dbadeb129a2617bb868f"
	return u.GroupsGetMembersCount(id)
}

func tgFollowers(id interface{}) (int, error) {
	bot, err := tgbotapi.NewBotAPI("419772144:AAGnsI8SrpBqSMUjL-ePwenqaPodpYCyckA")
	if err != nil {
		return 0, err
	}

	bot.Debug = false

	c := tgbotapi.ChatConfig{
	//ChatID: -1001133425345,
	//SuperGroupUsername: "@ughome",
	}
	switch realID := id.(type) {
	case string:
		c.SuperGroupUsername = "@" + realID
	case int:
		c.ChatID = int64(realID)
	}
	cnt, err := bot.GetChatMembersCount(c)
	if err != nil {
		return 0, err
	}

	return cnt, nil
}
