package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3" // 只导入，作为驱动

	"github.com/hoshinonyaruko/gensokyo-llm/applogic"
	"github.com/hoshinonyaruko/gensokyo-llm/config"
	"github.com/hoshinonyaruko/gensokyo-llm/hunyuan"
	"github.com/hoshinonyaruko/gensokyo-llm/template"
)

func main() {
	if _, err := os.Stat("config.yml"); os.IsNotExist(err) {

		// 将修改后的配置写入 config.yml
		err = os.WriteFile("config.yml", []byte(template.ConfigTemplate), 0644)
		if err != nil {
			fmt.Println("Error writing config.yml:", err)
			return
		}

		fmt.Println("请配置config.yml然后再次运行.")
		fmt.Print("按下 Enter 继续...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		os.Exit(0)
	}
	// 加载配置
	conf, err := config.LoadConfig("config.yml")
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	// Deprecated
	secretId := conf.Settings.SecretId
	secretKey := conf.Settings.SecretKey
	fmt.Printf("secretId:%v\n", secretId)
	fmt.Printf("secretKey:%v\n", secretKey)
	region := config.Getregion()
	client, err := hunyuan.NewClientWithSecretId(secretId, secretKey, region)
	if err != nil {
		fmt.Printf("创建hunyuanapi出错:%v", err)
	}

	db, err := sql.Open("sqlite3", "file:mydb.sqlite?cache=shared&mode=rwc")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	app := &applogic.App{DB: db, Client: client}
	// 在启动服务之前确保所有必要的表都已创建
	err = app.EnsureTablesExist()
	if err != nil {
		log.Fatalf("Failed to ensure database tables exist: %v", err)
	}
	// 确保user_context表存在
	err = app.EnsureUserContextTableExists()
	if err != nil {
		log.Fatalf("Failed to ensure user_context table exists: %v", err)
	}

	apiType := config.GetApiType() // 调用配置包的函数获取API类型

	switch apiType {
	case 0:
		// 如果API类型是0，使用app.chatHandlerHunyuan
		http.HandleFunc("/conversation", app.ChatHandlerHunyuan)
	case 1:
		// 如果API类型是1，使用app.chatHandlerErnie
		http.HandleFunc("/conversation", app.ChatHandlerErnie)
	case 2:
		// 如果API类型是2，使用app.chatHandlerChatGpt
		http.HandleFunc("/conversation", app.ChatHandlerChatgpt)
	default:
		// 如果是其他值，可以选择一个默认的处理器或者记录一个错误
		log.Printf("Unknown API type: %d", apiType)
	}

	http.HandleFunc("/gensokyo", app.GensokyoHandler)
	port := config.GetPort()
	portStr := fmt.Sprintf(":%d", port)
	fmt.Printf("listening on %v\n", portStr)
	// 这里阻塞等待并处理请求
	log.Fatal(http.ListenAndServe(portStr, nil))
}
