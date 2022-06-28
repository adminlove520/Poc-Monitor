package main

import (
	"encoding/json"
	"fmt"
	"github.com/sari3l/notify/notifier/bark"
	"github.com/sari3l/requests"
	nUrl "net/url"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 修改部分

// 查询关键字
const cveQuery = "CVE-20"

// 通知函数
var barkToken = os.Getenv("barkToken")

const barkGroup = "Poc-Monitor"

func Notice(updateItems *[]*Item) {
	for _, item := range *updateItems {
		webhook := fmt.Sprintf("https://api.day.app/%s/%s/%s", barkToken, nUrl.QueryEscape(item.Name), nUrl.QueryEscape(item.Description))
		option := bark.Option{Webhook: webhook}
		option.Url = item.HtmlUrl
		option.Group = barkGroup
		option.ToNotifier().Send(nil)
		fmt.Printf("[>] 新增 %s\n", webhook)
	}
}

// 以下勿动

var UpdateJsonFilePath = fmt.Sprintf("%s/update.json", GetCurrentDirectory())
var NewJsonFilePath = fmt.Sprintf("%s/new.json", GetCurrentDirectory())

var cveExp, _ = regexp.Compile(`(?i)CVE-(\d+)-\d+`)

func main() {
	currentYear := time.Now().Year()
	url := fmt.Sprintf("https://api.github.com/search/repositories?q=%s&sort=updated", cveQuery)
	resp := requests.Get(url)
	if resp == nil {
		fmt.Println(fmt.Errorf("[!] 无法访问 %s", url))
	}
	items := resp.Json().Get("items").Array()
	var addItems = make([]*Item, 0)
	var updateItems = make([]*Item, 0)
	for _, data := range items {
		item := new(Item)
		err := json.Unmarshal([]byte(data.Raw), item)
		if err != nil {
			continue
		}
		// 提取CVE信息
		cveInfo := cveExp.FindStringSubmatch(item.Name)
		// 查询本地信息
		if cveInfo == nil {
			continue
		}
		cveId, cveYear := strings.ToUpper(cveInfo[0]), cveInfo[1]
		if year, _ := strconv.Atoi(cveYear); year > currentYear {
			fmt.Println(fmt.Errorf("[!] 错误年限，%s", cveId))
			continue
		}
		cveYearPath := fmt.Sprintf("%s/%s", GetCurrentDirectory(), cveYear)
		cveFilePath := fmt.Sprintf("%s/%s/%s.json", GetCurrentDirectory(), cveYear, cveId)
		// 检查年限
		if !CheckFileExists(cveYearPath) {
			if err = CreateDir(cveYearPath); err != nil {
				fmt.Println(fmt.Errorf("[!] 创建 %s 失败, %s", cveYearPath, err))
				continue
			}
		}
		// 检查cve信息
		var historyItems = make([]*Item, 0)
		if CheckFileExists(cveFilePath) {
			// 读取历史cve信息
			err = ReadJsonFile(cveFilePath, &historyItems)
			if err != nil && err != EmptyError {
				fmt.Println(fmt.Errorf("[!] 读取 %s 失败, %s", cveFilePath, err))
			}
		}
		needAdd := true
		for index, historyItem := range historyItems {
			if item.Id == historyItem.Id {
				// diff
				if !reflect.DeepEqual(*item, *historyItem) {
					itemsContentValues := historyItems
					itemsContentValues[index] = item
					updateItems = append(updateItems, item)
					fmt.Printf("[>] 更新 %s %d\n", cveId, item.Id)
				}
				needAdd = false
				break
			}
		}
		if needAdd {
			historyItems = append(historyItems, item)
			addItems = append(addItems, item)
		}
		// 更新cve信息
		byteValue, err := json.Marshal(historyItems)
		if err != nil {
			fmt.Println(fmt.Errorf("[!] 转换 %s 内容失败, %s", cveId, err))
		}
		if err = WriteFile(cveFilePath, byteValue); err != nil {
			fmt.Println(fmt.Errorf("[!] 写入 %s 内容失败, %s", cveFilePath, err))
		}
	}

	if len(addItems) != 0 || len(updateItems) != 0 {
		// 更新dateLog
		logPath := fmt.Sprintf("%s/%s", GetCurrentDirectory(), LogFilePath)
		if !CheckFileExists(logPath) {
			_ = CreateDir(logPath)
		}
		dateLogFilePath := fmt.Sprintf("%s/%s.json", logPath, time.Now().Format("2006-01-02"))
		dateLogItems := DateLog{}
		if CheckFileExists(dateLogFilePath) {
			// 读取历史cve信息
			err := ReadJsonFile(dateLogFilePath, &dateLogItems)
			if err != nil && err != EmptyError {
				fmt.Println(fmt.Errorf("[!] 读取 %s 失败, %s", dateLogFilePath, err))
			}
		}
		dateLogItems.New = append(dateLogItems.New, addItems...)
		for _, item := range updateItems {
			for logIndex, logItem := range dateLogItems.Update {
				if item.Id == logItem.Id {
					if !reflect.DeepEqual(*item, *logItem) {
						dateLogItems.Update[logIndex] = item
					}
					break
				}
			}
			dateLogItems.Update = append(dateLogItems.Update, item)
		}
		byteValue, err := json.Marshal(dateLogItems)
		if err != nil {
			fmt.Println(fmt.Errorf("[!] 转换日志内容失败, %s", err))
		}
		if err = WriteFile(dateLogFilePath, byteValue); err != nil {
			fmt.Println(fmt.Errorf("[!] 写入日志内容失败, %s", err))
		}

		// 更新new/update内容
		if len(updateItems) != 0 {
			byteValue, err = json.Marshal(updateItems)
			if err != nil {
				fmt.Println(fmt.Errorf("[!] 转换更新内容失败, %s", err))
			}
			if err = WriteFile(UpdateJsonFilePath, byteValue); err != nil {
				fmt.Println(fmt.Errorf("[!] 写入更新内容失败, %s", err))
			}
		}
		if len(addItems) != 0 {
			byteValue, err = json.Marshal(addItems)
			if err != nil {
				fmt.Println(fmt.Errorf("[!] 转换新增内容失败, %s", err))
			}
			if err = WriteFile(NewJsonFilePath, byteValue); err != nil {
				fmt.Println(fmt.Errorf("[!] 写入新增内容失败, %s", err))
			}
			// 新增后通知
			Notice(&addItems)
		}
	}
}
