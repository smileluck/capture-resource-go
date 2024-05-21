package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"log"

	"github.com/glebarez/sqlite"
	"github.com/tidwall/gjson"
	"gorm.io/gorm"
)

var globalDb *gorm.DB = connect("temp.db")

func connect(path string) (db *gorm.DB) {
	if !fileExists(path) {
		file, err := os.Create(path)
		if err != nil {
			fmt.Println("Error creating file:", err)
			return
		}
		defer file.Close() // 确保文件在函数结束时关闭
		fmt.Printf("File %s created successfully\n", path)
	}

	db, err2 := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err2 != nil {
		panic("failed to connect database")
	}
	return db
}
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

type Video struct {
	ID               uint `gorm:"primarykey;autoIncrement"`
	source_id        int64
	source_type      string
	title            string
	tags             string
	description      string
	origin_video_url string
	origin_cover_url string
	width            int64
	height           int64
	duration         int64
	req_body         string
	ai_generate_flag int64
	bucket_flag      uint8
	target_video_key string
	target_cover_url string
	target_width     int64
	target_height    int64
	create_time      time.Time
	save_time        time.Time
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Println("Please choice item:")
		fmt.Println("1: 开始爬取数据")
		fmt.Println("2: 退出程序")
		text, _, err := reader.ReadLine()
		if err != nil {
			panic(fmt.Errorf("发生致命错误: %w \n", err))
		}
		if string(text) == "1" {
			go capture()
		} else {
			break
		}
	}
}

func capture() {
	// 异常捕获
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("Panicing %s\r\n", e)
		}
	}()
	globalDb.AutoMigrate(&Video{})

	pagis := 1
	page, err := capturePixabay(1)
	if err == nil {
		pagis = int(page.Get("pages").Int())
		total := int(page.Get("total").Int())
		fmt.Println("pages=>", pagis, "total=>", total)

		for pagi := 1; pagi <= pagis; pagi++ {
			page, err := capturePixabay(pagi)
			if err == nil {
				results := page.Get("results").Array()

				for _, v := range results {
					description := v.Get("description")
					duration := v.Get("duration").Int()
					height := v.Get("height").Int()
					width := v.Get("width").Int()
					name := v.Get("name").Str
					alt := v.Get("alt")

					isAiGenerated := v.Get("isAiGenerated").Int()
					mp4 := v.Get("source.mp4")
					thumbnail := v.Get("source.thumbnail")
					source_id := v.Get("id").Int()

					record := Video{
						source_id:        source_id,
						source_type:      "pixabay",
						title:            name,
						tags:             alt.Str,
						description:      description.Str,
						origin_video_url: mp4.Str,
						origin_cover_url: thumbnail.Str,
						width:            width,
						height:           height,
						duration:         duration,
						req_body:         v.Str,
						ai_generate_flag: isAiGenerated,
						bucket_flag:      0,
						// target_video_key: "",
						// target_cover_url: "",
						// target_width:     0,
						// target_height:    0,
						create_time: time.Now(),
					}
					globalDb.Select(
						"source_id",
						"source_type",
						"title",
						"tags",
						"description",
						"origin_video_url",
						"origin_cover_url",
						"width",
						"height",
						"duration",
						"req_body",
						"ai_generate_flag",
						"bucket_flag", "create_time").Create(&record)
				}

			}
		}
	}
}

// type PixabayRes struct {
// 	Page PixabayPageModel `json:"page"`
// }
// type PixabayPageModel struct {
// 	MediaType       string                   `json:"mediaType"`
// 	Pages           string                   `json:"pages"`
// 	Total           string                   `json:"total"`
// 	Results         []PixabayPageResultModel `json:"results"`
// 	IsAiGenerated   bool                     `json:"isAiGenerated"`
// 	IsEditorsChoice bool                     `json:"isEditorsChoice"`
// 	IsLowQuality    bool                     `json:"isLowQuality"`
// 	Description     string                   `json:"description"`
// 	name            string                   `json:"name"`
// }

// type PixabayPageResultModel struct {
// }

func capturePixabay(pagi int) (result gjson.Result, err error) {
	url := "https://pixabay.com/zh/videos/search/?order=ec&pagi="
	resp, err := http.Get(url + fmt.Sprintf("%d", pagi))

	if err != nil {
		log.Println("request error : ", err)
		return result, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("read response body failed:", err)
		return result, err
	}
	fmt.Println("http get succeed:", string(respBody))

	return gjson.Get(string(respBody), "page"), nil
}
