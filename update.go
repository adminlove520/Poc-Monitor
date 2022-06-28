package main

import (
	"encoding/json"
	"fmt"
	"github.com/sari3l/requests"
	"github.com/sari3l/requests/ext"
	"io/ioutil"
	"strings"
)

const metadataFileName = "metadata.json"

type Metadata struct {
	Published string `json:"Published"`
	Summary   string `json:"summary"`
}

func main() {
	var err error
	rootDir, _ := ioutil.ReadDir(GetCurrentDirectory())
	for _, dir := range rootDir {
		if !(dir.IsDir() && strings.HasPrefix(dir.Name(), "20")) { // 需要修改
			continue
		}
		var markdown string
		metadataMap := make(map[string]Metadata)
		// 读取历史metadata
		metadataFilePath := fmt.Sprintf("%s/%s/%s", GetCurrentDirectory(), dir.Name(), metadataFileName)
		metadataUpdate := false
		if CheckFileExists(metadataFilePath) {
			ReadJsonFile(metadataFilePath, &metadataMap)
		}
		yearDir, _ := ioutil.ReadDir(fmt.Sprintf("%s/%s", GetCurrentDirectory(), dir.Name()))
		for _, cveFile := range yearDir {
			if !strings.HasPrefix(cveFile.Name(), "CVE") {
				continue
			}
			var unit string
			items := make([]*Item, 0)
			cveId := strings.ToUpper(strings.Split(cveFile.Name(), ".json")[0])
			// 获取metadata
			summary := ""
			published := ""
			title := cveId
			if metadataMap[cveId].Summary != "" {
				summary = metadataMap[cveId].Summary
				published = metadataMap[cveId].Published
			} else {
				fmt.Println(fmt.Sprintf("[>] 正在请求 %s 数据", cveId))
				resp := requests.Get(fmt.Sprintf("https://cve.circl.lu/api/cve/%s", cveId), ext.Timeout(60))
				if resp != nil && resp.Ok && resp.Content != "null" {
					result := resp.Json()
					summary = result.Get("summary").Str
					published = result.Get("Published").Str
					metadataMap[cveId] = Metadata{
						Summary:   summary,
						Published: published,
					}
					metadataUpdate = true
				}
			}
			title = fmt.Sprintf("%s (%s)", title, published)
			unit += fmt.Sprintf("## %s\n", title)
			unit += fmt.Sprintf("> %s\n", summary)
			ReadJsonFile(fmt.Sprintf("%s/%s/%s", GetCurrentDirectory(), dir.Name(), cveFile.Name()), &items)
			for _, item := range items {
				unit += fmt.Sprintf("- [%s](%s)		<img src=\"https://user-images.githubusercontent.com/45752995/176187524-029a1a42-1c31-4d23-823a-989c40ca8460.svg\" alt=\"fork\"/>%d <img src=\"https://user-images.githubusercontent.com/45752995/176188923-7eb4772f-794e-48da-962b-5f8f69ca184f.svg\" alt=\"star\"/>%d\n", item.FullName, item.HtmlUrl, item.ForksCount, item.StargazersCount)
			}
			markdown = fmt.Sprintf("\n---\n%s", unit) + markdown
		}
		markdown = fmt.Sprintf("# %s List\n", dir.Name()) + markdown
		readMePath := fmt.Sprintf("%s/%s/README.md", GetCurrentDirectory(), dir.Name())
		markdownHistory := string(ReadFile(readMePath))
		if markdownHistory != markdown {
			fmt.Println(fmt.Sprintf("[>] 正在更新 %s README.md", dir.Name()))
			err = WriteFile(readMePath, []byte(markdown))
			if err != nil {
				fmt.Println(fmt.Errorf("[!] 写入 %s README.md 失败 %s", dir.Name(), err))
			}
		}
		if metadataUpdate {
			fmt.Println(fmt.Sprintf("[>] 正在更新 %s metadata", dir.Name()))
			byteValue, _ := json.Marshal(metadataMap)
			err = WriteFile(metadataFilePath, byteValue)
			if err != nil {
				fmt.Println(fmt.Errorf("[!] 更新 %s metadata 失败 %s", dir.Name(), err))
			}
		}
	}
}
