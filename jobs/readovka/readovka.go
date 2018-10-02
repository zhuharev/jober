package readovka

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	ini "gopkg.in/ini.v1"

	"github.com/PuerkitoBio/goquery"
	"github.com/Unknwon/com"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/toby3d/telegraph"
	dry "github.com/ungerik/go-dry"
	"github.com/zhuharev/intarr"
	"github.com/zhuharev/jober/jobs"
	"github.com/zhuharev/jober/types"
)

var (
	fName = "data/readovka/readovkaLastID"
	debug = true
)

func init() {
	jobs.Register("readovka", New())
}

func logit(format string, v ...interface{}) {
	if debug {
		log.Printf(format, v...)
	}
}

type Robot struct {
	token   string
	bot     *tgbotapi.BotAPI
	adminID int64
}

func New() *Robot {
	return &Robot{}
}

func (r *Robot) Init(cfg *ini.Section) (types.Job, error) {
	err := os.MkdirAll("data/readovka", 0777)
	if err != nil {
		return r, err
	}

	bot, err := tgbotapi.NewBotAPI("284214846:AAF9VIULJLrx42TaZ295hoqYt7a1l47EJg4")
	if err != nil {
		log.Panic(err)
	}

	r.bot = bot

	return r, nil
}

func (r *Robot) Run() (err error) {
	for {
		log.Println("job readovka")
		err = r.job()
		if err != nil {
			log.Println("Err readovka job:", err)
		}
		time.Sleep(5 * time.Minute)
	}
}

func (r *Robot) job() error {
	lID, err := getLastID()
	if err != nil {
		return err
	}
	logit("last id: %d", lID)
	ids, err := getNews(lID)
	if err != nil {
		return err
	}
	logit("getted news: %+v", ids)
	for _, id := range ids {
		// if i > 20 {
		// 	return nil
		// }
		err = r.sendNew(id)
		if err != nil {
			return err
		}
		err = setLastID(id)
		if err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

func getLastID() (uint64, error) {
	str, _ := dry.FileGetString(fName)
	str = strings.TrimSpace(str)
	return uint64(com.StrTo(str).MustInt()), nil
}

func setLastID(lID uint64) error {
	return dry.FileSetString(fName, fmt.Sprint(lID))
}

func (r *Robot) sendNew(id uint64) error {

	title, body, publ, err := getTitle(id)
	if err != nil {
		return err
	}

	if time.Since(publ) > time.Hour*24 {
		logit("skip old new: %d (%s)", id, publ)
		return nil
	}

	logit("title: %s\nbody: %s", title, body)

	account := &telegraph.Account{
		ShortName:   "Sandbox",
		AuthorName:  "Anonymous",
		AccessToken: "b8e3a67b884fb628d23575764831697fa0d2b52b73026075ca04bd9580c0",
		AuthURL:     "https://edit.telegra.ph/auth/HS33dG8C8NU3ZhaM2aIjAAxYPqFSiB1Xt21EI7gOU9",
	}

	content, err := telegraph.ContentFormat(body)
	if err != nil {
		panic(err)
	}

	logit("create telegraph page...")
	page, err := account.CreatePage(&telegraph.Page{
		Title:      title,
		AuthorName: account.AuthorName,
		Content:    content,
	}, true)
	if err != nil {
		log.Printf("err create page on telegraph: %s", err)
	}

	//strID := fmt.Sprint(id)

	text := fmt.Sprintf("*%s*\n\n[Instant View](%s)\n",
		title,
		page.URL,
	)

	msg := tgbotapi.NewMessageToChannel("@readovka", text)
	msg.DisableWebPagePreview = false
	msg.DisableNotification = true

	msg.ParseMode = "Markdown"

	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("Читать на сайте", fmt.Sprintf("https://readovka.ru/news/%d", id)),
		),
	)
	msg.ReplyMarkup = &markup

	_, err = r.bot.Send(msg)
	return err

}

func getNews(lastNew uint64) (ids intarr.Slice, err error) {
	bts, err := fetch("https://readovka.ru/")
	if err != nil {
		log.Println(err)
	}

	re := regexp.MustCompile(`/news/([\d]+)`)
	arr := re.FindAllSubmatch(bts, -1)

	for _, v := range arr {
		if len(v) > 1 {
			if num := com.StrTo(string(v[1])).MustInt64(); uint64(num) > lastNew && !ids.In(uint64(num)) {
				ids = append(ids, uint64(num))
			}
		}
	}

	ids.Sort()
	return
}

func getTitle(id uint64) (title string, body string, publ time.Time, err error) {
	u := fmt.Sprintf("https://readovka.ru/news/%d", id)
	data, err := fetch(u)
	if err != nil {
		return
	}
	rdr := bytes.NewReader(data)
	doc, err := goquery.NewDocumentFromReader(rdr)
	if err != nil {
		return
	}
	title = doc.Find(".block-news h1").Eq(0).Text()
	body, err = doc.Find(".block-fullnews").Html()
	if err != nil {
		return
	}
	publStr, has := doc.Find(".block-news time").Attr("datetime")
	if !has {
		logit("not found publ time: %d", id)
	}
	publ, err = time.Parse(time.RFC3339, publStr)
	if err != nil {
		logit("err parse time: %s", err)
	}
	return
}

func fetch(url string) ([]byte, error) {
	logit("fetch %s", url)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalln(err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/60.0.3112.113 Safari/537.36")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	return body, nil
}
