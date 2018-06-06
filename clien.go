package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	resty "gopkg.in/resty.v0"
)

type clienClient struct {
	client *resty.Client
	logged bool
	userID string
}

func newClienClient() *clienClient {
	restyClient := resty.New()
	restyClient.SetHTTPMode()
	client := &clienClient{
		client: restyClient,
	}
	return client
}

func (c *clienClient) CSRFToken(url string) (token string, err error) {
	if url == "" {
		url = "https://m.clien.net/service/auth/login"
	}
	var resp *resty.Response
	resp, err = c.client.R().Get(url)
	if err != nil {
		return
	}

	r := bytes.NewReader(resp.Body())
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return
	}

	token, _ = doc.Find("input[name=_csrf]").First().Attr("value")
	return token, nil
}

func (c *clienClient) Login(token, username, password string) (succeed bool) {
	req := c.client.R()
	req.SetFormData(map[string]string{
		"_csrf":        token,
		"userId":       username,
		"userPassword": password,
	})
	req.SetHeader("Referer", "https://m.clien.net/service/auth/login")
	resp, err := req.Post("https://m.clien.net/service/login")
	if err != nil {
		log.Println(err)
		return
	}

	r := bytes.NewReader(resp.Body())
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		log.Println(err)
		return
	}
	if fail := doc.Find("div.side-account.after").Length(); fail > 0 {
		return
	}

	c.logged = true
	c.loadMyInfo()
	return c.logged
}

func (c *clienClient) loadMyInfo() bool {
	if !c.logged {
		return false
	}

	req := c.client.R()
	resp, err := req.Get("https://m.clien.net/service/mypage/myInfo")
	if err != nil {
		log.Println(err)
		return false
	}

	r := bytes.NewReader(resp.Body())
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		log.Println(err)
		return false
	}
	c.userID = doc.Find("#myInfoForm table tr:first-child td:last-child").First().Text()
	log.Printf("verified username: %s", c.userID)
	return true
}

func (c *clienClient) Articles() (list []*clienArticle) {
	baseURL := "https://m.clien.net/service/mypage/myArticle?&type=articles&po=%d&ps=100"
	for i := 0; ; i += 1 {
		url := fmt.Sprintf(baseURL, i)
		req := c.client.R()
		resp, err := req.Get(url)
		if err != nil {
			log.Println(err)
			return
		}

		r := bytes.NewReader(resp.Body())
		doc, err := goquery.NewDocumentFromReader(r)
		if err != nil {
			log.Println(err)
			return
		}
		if empty := doc.Find("div.list-empty.line").Length(); empty > 0 {
			log.Println("Empty table found")
			break
		}
		rows := doc.Find("div.board-list div.list-row")
		log.Printf("%d article(s) found in page %d", rows.Length(), i+1)

		rows.Each(func(i int, s *goquery.Selection) {
			link := s.Find("a.list-subject").First()
			href, _ := link.Attr("href")
			tmp := strings.Split(href, "/")
			articleID, _ := strconv.Atoi(tmp[len(tmp)-1])
			title := strings.TrimSpace(link.Text())
			article := &clienArticle{
				ID:     articleID,
				Title:  title,
				client: c,
			}
			list = append(list, article)
			log.Printf("found article ID %d: %s", articleID, title)
		})

		time.Sleep(time.Millisecond * 100)
	}

	return
}

func (c *clienClient) Comments() (list []*clienComment) {
	baseURL := "https://m.clien.net/service/mypage/myArticle?&type=comments&po=%d&ps=100"
	for i := 0; ; i += 1 {
		url := fmt.Sprintf(baseURL, i)
		req := c.client.R()
		resp, err := req.Get(url)
		if err != nil {
			log.Println(err)
			return
		}

		r := bytes.NewReader(resp.Body())
		doc, err := goquery.NewDocumentFromReader(r)
		if err != nil {
			log.Println(err)
			return
		}
		if empty := doc.Find("div.list-empty.line").Length(); empty > 0 {
			log.Println("Empty table found")
			break
		}
		rows := doc.Find("div.board-list div.list-row")
		log.Printf("%d comment(s) found in page %d", rows.Length(), i+1)

		rows.Each(func(i int, s *goquery.Selection) {
			link := s.Find("a.list-subject").First()
			href, _ := link.Attr("href")
			tmp := strings.Split(href, "/")
			articleID, _ := strconv.Atoi(tmp[len(tmp)-1])
			boardID := tmp[len(tmp)-2]
			title := strings.TrimSpace(link.Text())
			comment := &clienComment{
				ArticleID: articleID,
				BoardID:   boardID,
				Summary:   title,
				client:    c,
			}
			loaded := comment.LoadCommentID(&list)
			if loaded {
				list = append(list, comment)
			}
			log.Printf("found article ID %d comment ID %d: %s", articleID, comment.CommentID, title)
		})

		time.Sleep(time.Millisecond * 50)
	}

	return
}

type clienArticle struct {
	ID     int
	Title  string
	client *clienClient
}

type clienComment struct {
	ArticleID int
	BoardID   string
	CommentID int
	Summary   string
	client    *clienClient
}

type clienCommentItem struct {
	CommentID int `json:"commentSn"`
	Member    struct {
		UserID string `json:"userId"`
	} `json:"member"`
}

func (a *clienArticle) Delete() bool {

	return true
}

func (c *clienComment) LoadCommentID(list *[]*clienComment) bool {
	baseURL := "https://m.clien.net/service/api/board/%s/%d/comment"
	params := `{"order":"date","po":%d,"ps":9999}`

	url := fmt.Sprintf(baseURL, c.BoardID, c.ArticleID)
	for i := 0; ; i += 1 {
		payload := fmt.Sprintf(params, i)
		req := c.client.client.R()
		req.SetQueryParam("param", payload)
		resp, err := req.Get(url)
		if err != nil {
			log.Println(err)
			continue
		}

		var comments []clienCommentItem
		err = json.Unmarshal(resp.Body(), &comments)
		if err != nil {
			log.Println(err)
			continue
		}
		if len(comments) == 0 {
			break
		}

		for _, comment := range comments {
			if comment.Member.UserID == c.client.userID {
				dup := false
				for _, item := range *list {
					if item.CommentID == 0 {
						continue
					} else if item.CommentID == comment.CommentID {
						continue
					}
				}

				if dup {
					continue
				}
				c.CommentID = comment.CommentID
				break
			}
		}
	}
	return c.CommentID > 0
}

func (c *clienComment) Delete() bool {
	if c.CommentID == 0 {
		return false
	}
	csrf, _ := c.client.CSRFToken(fmt.Sprintf("https://m.clien.net/service/board/%s/%d", c.BoardID, c.ArticleID))
	url := fmt.Sprintf("https://m.clien.net/service/api/board/%s/%d/comment/delete/%d", c.BoardID, c.ArticleID, c.CommentID)
	req := c.client.client.R()
	req.SetHeader("X-Requested-With", "XMLHttpRequest")
	req.SetHeader("X-CSRF-TOKEN", csrf)
	resp, err := req.Post(url)
	if err != nil {
		log.Println(err)
		return false
	}

	return resp.String() == "true"
}
