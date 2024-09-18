package main

import (
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/charset"
	"github.com/emersion/go-message/mail"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Mail struct {
	Name     string `mapstructure:"name"`
	Password string `mapstructure:"password"`
	Server   string `mapstructure:"server"`
}

var Logger *zap.Logger

func main() {
	router := gin.Default()

	// 添加 CORS 中间件
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	router.POST("/api/v1/getPassword", GetPassword)

	router.Run(":18080")
}

func GetPassword(c *gin.Context) {
	var data struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	a := Mail{
		Name:     data.Username,
		Password: data.Password,
		Server:   "imap.exmail.qq.com:993",
	}

	if !strings.Contains(a.Server, "imap.") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Server must be an IMAP server"})
	}

	secr, err := Usage(a)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	encodedPassword := base64.StdEncoding.EncodeToString([]byte(secr))

	c.JSON(http.StatusOK, gin.H{"result": encodedPassword})
}

// CustomerImapClient 调用NewImapClient
func CustomerImapClient(name, password, server string) (*client.Client, error) {
	// 【修改】账号和密码
	return NewImapClient(name, password, server)
}

// NewImapClient 创建IMAP客户端
func NewImapClient(username, password, server string) (*client.Client, error) {
	// 【字符集】  处理us-ascii和utf-8以外的字符集(例如gbk,gb2313等)时,
	//  需要加上这行代码。
	// 【参考】 https://github.com/emersion/go-imap/wiki/Charset-handling
	imap.CharsetReader = charset.Reader

	//Logger.Sugar().Info("Connecting to server...")

	// 连接邮件服务器
	c, err := client.DialTLS(server, nil)
	if err != nil {
		Logger.Sugar().Fatal(err)
		return nil, err
	}
	//Logger.Sugar().Info("Connected")

	// 使用账号密码登录
	if err := c.Login(username, password); err != nil {
		return nil, err
	}

	//Logger.Sugar().Info("Logged in")

	return c, nil
}

func Usage(cmail Mail) (pass string, err error) {
	// 连接邮件服务器
	c, err := CustomerImapClient(cmail.Name, cmail.Password, cmail.Server)
	if err != nil {
		return "", err
	}

	// 查看有什么邮箱
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() {
		done <- c.List("", "*", mailboxes)
	}()

	for m := range mailboxes {
		slog.Info(m.Name)
	}

	if err := <-done; err != nil {
		slog.Info(err.Error())
	}

	// 选择收件箱
	_, err = c.Select("INBOX", false)
	if err != nil {
		slog.Info(err.Error())
	}

	// 搜索条件实例对象
	criteria := imap.NewSearchCriteria()

	// ALL是默认条件
	// See RFC 3501 section 6.4.4 for a list of searching criteria.
	criteria.WithoutFlags = []string{"ALL"}
	ids, _ := c.Search(criteria)
	var s imap.BodySectionName

	for {
		if len(ids) == 0 {
			break
		}
		id := pop(&ids)

		seqset := new(imap.SeqSet)
		seqset.AddNum(id)
		chanMessage := make(chan *imap.Message, 1)
		go func() {
			// 第一次fetch, 只抓取邮件头，邮件标志，邮件大小等信息，执行速度快
			if err = c.Fetch(seqset,
				[]imap.FetchItem{imap.FetchEnvelope, imap.FetchFlags, imap.FetchRFC822Size},
				chanMessage); err != nil {
				// 【实践经验】这里遇到过的err信息是：ENVELOPE doesn't contain 10 fields
				// 原因是对方发送的邮件格式不规范，解析失败
				// 相关的issue: https://github.com/emersion/go-imap/issues/143
				//Logger.Sugar().Info(seqset, err)
			}
		}()

		message := <-chanMessage
		if message == nil {
			//Logger.Sugar().Info("Server didn't returned message")
			continue
		}
		//Logger.Sugar().Infof("%v: %v bytes, flags=%v \n", message.SeqNum, message.Size, message.Flags)

		if strings.HasPrefix(message.Envelope.Subject, "EB VPN Password") {
			chanMsg := make(chan *imap.Message, 1)
			go func() {
				// 这里是第二次fetch, 获取邮件MIME内容
				if err = c.Fetch(seqset,
					[]imap.FetchItem{imap.FetchRFC822},
					chanMsg); err != nil {
					slog.Info(seqset.String())
					slog.Info(err.Error())
				}
			}()

			msg := <-chanMsg
			if msg == nil {
				slog.Info("Server didn't returned message")
			}

			section := &s
			r := msg.GetBody(section)
			if r == nil {
				slog.Info("Server didn't returned message body")
			}

			// Create a new mail reader
			// 创建邮件阅读器
			mr, err := mail.CreateReader(r)
			if err != nil {
				slog.Info(err.Error())
			}

			// Process each message's part
			// 处理消息体的每个part
			for {
				p, err := mr.NextPart()
				if err == io.EOF {
					break
				} else if err != nil {
					slog.Info(err.Error())
				}

				switch p.Header.(type) {
				case *mail.InlineHeader:
					// This is the message's text (can be plain-text or HTML)
					// 获取正文内容, text或者html
					b, _ := io.ReadAll(p.Body)
					slog.Info(string(b))

					// 定义正则表达式模式
					pattern := `Your password: (\w+)`

					// 编译正则表达式
					re := regexp.MustCompile(pattern)

					// 查找匹配项
					matches := re.FindStringSubmatch(string(b))

					// 如果找到匹配项，则输出密码后面的字符串并写入文件
					if len(matches) > 1 {
						password := matches[1]
						pass = password
					} else {
						slog.Info("Password not found.")
					}
				}
				slog.Info("已找到满足需求的邮件")
				return pass, nil
			}
		}
	}

	return pass, nil
}

func pop(list *[]uint32) uint32 {
	length := len(*list)
	lastEle := (*list)[length-1]
	*list = (*list)[:length-1]
	return lastEle
}
