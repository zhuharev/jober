package vkim

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	ini "gopkg.in/ini.v1"

	"github.com/Unknwon/com"
	"github.com/zhuharev/jober/jobs"
	"github.com/zhuharev/jober/types"
	"github.com/zhuharev/vkutil"

	vk "github.com/urShadow/go-vk-api"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

var (
	bName       = []byte("vkim")
	vkNameCache = map[int64]VkUser{}

	targets = map[int64]int64{}
)

type VkUser struct {
	ID         int64
	FirstName  string
	LastName   string
	ScreenName string
}

func init() {
	jobs.Register("vkim", New())

}

type Robot struct {
	token     string
	bot       *tgbotapi.BotAPI
	adminID   int64
	userToken string

	vkPullApi *vk.VK

	inited bool
}

func New() *Robot {
	r := &Robot{}
	return r
}

func (r *Robot) Init(cfg *ini.Section) (j types.Job, err error) {
	r.token = cfg.Key("token").String()
	r.adminID = cfg.Key("admin_id").MustInt64()
	r.userToken = cfg.Key("user_token").String()
	r.inited = true

	log.Println("Start im bot", r.token)
	r.bot, err = tgbotapi.NewBotAPIWithClient(r.token, &http.Client{})
	if err != nil {

		log.Println(err)
		return r, nil
	}

	r.vkPullApi = vk.New("ru")
	err = r.vkPullApi.Init(r.userToken)
	if err != nil {
		log.Println(err)
		return
	}
	log.Println(r.vkPullApi, r.userToken)

	msg := tgbotapi.NewMessage(r.adminID, "Vkim bot started")
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

	vku := vkutil.NewWithToken(r.userToken)
	var msgFmt = `*%s %s (%s):*
	%s`

	log.Println(r.inited, r.vkPullApi)
	go func(r *Robot) {
		if r.vkPullApi == nil {
			log.Println("nil")
		}
		r.vkPullApi.OnNewMessage(func(msg *vk.LPMessage) {
			if msg == nil {
				return
			}
			if msg.Flags&vk.FlagMessageOutBox == 0 {
				// if msg.Text == "/hello" {
				// 	api.Messages.Send(vk.RequestParams{
				// 		"peer_id":          strconv.FormatInt(msg.FromID, 10),
				// 		"message":          "Hello!",
				// 		"forward_messages": strconv.FormatInt(msg.ID, 10),
				// 	})
				// }

				user, has := vkNameCache[msg.FromID]
				if !has {
					users, err := vku.UsersGet(msg.FromID, url.Values{"fields": {"screen_name"}})
					if err != nil {
						log.Println(err)
						return
					}
					if len(users) != 1 {
						return
					}
					vkUser := users[0]

					user.FirstName = vkUser.FirstName
					user.LastName = vkUser.LastName
					user.ScreenName = vkUser.ScreenName
					user.ID = msg.FromID
					vkNameCache[msg.FromID] = user
				}

				tmsg := tgbotapi.NewMessage(102710272, fmt.Sprintf(msgFmt,
					user.FirstName, user.LastName, user.ScreenName, msg.Text))
				tmsg.ParseMode = tgbotapi.ModeMarkdown

				markup := tgbotapi.NewInlineKeyboardMarkup()
				button := tgbotapi.NewInlineKeyboardButtonData("Ответить", fmt.Sprintf("%d", msg.FromID))
				markup.InlineKeyboard = append(markup.InlineKeyboard, tgbotapi.NewInlineKeyboardRow(button))

				tmsg.ReplyMarkup = &markup
				_, err = r.bot.Send(tmsg)
				if err != nil {
					log.Println(err)
				}
			}
		})
		log.Println("here")
		r.vkPullApi.RunLongPoll()
	}(r)

	for update := range updates {
		if update.Message == nil {
			if update.CallbackQuery != nil {
				err = r.handleCallback(update.CallbackQuery)
				if err != nil {
					log.Println(err)
				}
			}
			continue
		}
		err = r.handler(update.Message)
		if err != nil {
			log.Println(err)
		}
	}
	return nil
}

func (r *Robot) vkUserID(tgUserID int64) (id int, err error) {
	return
}

func (r *Robot) lastMessageID(vkUserID int) (id int, err error) {
	return
}

func (r *Robot) handler(m *tgbotapi.Message) error {

	id, has := targets[int64(m.From.ID)]
	if !has {
		_, err := r.bot.Send(tgbotapi.NewMessage(m.Chat.ID, "Некому отправлять :("))
		if err != nil {
			log.Println(err)
		}
		return nil
	}

	_, err := r.vkPullApi.Messages.Send(vk.RequestParams{
		"peer_id": fmt.Sprint(id),
		"message": m.Text,
	})

	delete(targets, int64(m.From.ID))

	//msg := tgbotapi.NewMessage(m.Chat.ID, "lol")
	//_, err := r.bot.Send(msg)
	return err
}
func (r *Robot) handleCallback(m *tgbotapi.CallbackQuery) error {
	id := com.StrTo(m.Data).MustInt64()
	targets[m.Message.Chat.ID] = id
	_, err := r.bot.Send(tgbotapi.NewMessage(m.Message.Chat.ID, "Введите ответное сообщение:"))
	if err != nil {
		log.Println(err)
	}
	return nil
}
