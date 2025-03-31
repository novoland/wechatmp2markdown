package server

import (
	"encoding/json"
	// "fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"os"
	"path/filepath"

	"github.com/fengxxc/wechatmp2markdown/format"
	"github.com/fengxxc/wechatmp2markdown/parse"
	// "github.com/fengxxc/wechatmp2markdown/util"
)

func Start(addr string) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// rawQuery := r.URL.RawQuery
        paramsMap := make(map[string]string)

		if r.Method == "POST" {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				log.Printf("Error reading body: %v", err)
			} else {
				// 打印请求体内容
				fmt.Printf("Request body: %s\n", string(body))
				bodyParams, err := url.ParseQuery(string(body))
				if err == nil {
					for k, v := range bodyParams {
						// 打印键值对
						fmt.Printf("Key: %s, Value: %v\n", k, v)
						if len(v) > 0 {
							paramsMap[k] = v[0]
						}
					}
				}
			}
		}

		// url param
		wechatmpURL := paramsMap["url"]
		fmt.Printf("accept url: %s\n", wechatmpURL)
		imageArgValue := paramsMap["image"]
		fmt.Printf("     image: %s\n", imageArgValue)
		imagePolicy := parse.ImageArgValue2ImagePolicy(imageArgValue)

		if wechatmpURL == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(defHTML))
			return
		}
		// w.Header().Set("Content-Type", "application/octet-stream")
		// var articleStruct parse.Article = parse.ParseFromURL(wechatmpURL, imagePolicy)
		// title := articleStruct.Title.Val.(string)
		// mdString, saveImageBytes := format.Format(articleStruct)
		// if len(saveImageBytes) > 0 {
		// 	w.Header().Set("Content-Disposition", "attachment; filename="+title+".zip")
		// 	saveImageBytes[title] = []byte(mdString)
		// 	util.HttpDownloadZip(w, saveImageBytes)
		// } else {
		// 	w.Header().Set("Content-Disposition", "attachment; filename="+title+".md")
		// 	w.Write([]byte(mdString))
		// }


		w.Header().Set("Content-Type", "application/json")
		var articleStruct parse.Article = parse.ParseFromURL(wechatmpURL, imagePolicy)
		title := articleStruct.Title.Val.(string)
		mdString, saveImageBytes := format.Format(articleStruct)
		
		response := map[string]interface{}{
			"title":   title,
			"content": mdString,
		}
		if len(saveImageBytes) > 0 {
			response["has_images"] = true
		}
		jsonData, err := json.Marshal(response)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "failed to marshal json"}`))
			return
		}
		w.Write(jsonData)
	})

	// 图片服务处理
http.HandleFunc("/img", func(w http.ResponseWriter, r *http.Request) {
	imageName := r.URL.Query().Get("name")
	if imageName == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "image name is required"}`))
		return
	}

	imagePath := filepath.Join("/root/img", imageName)
	imageData, err := os.ReadFile(imagePath)
	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "image not found"}`))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "failed to read image"}`))
		}
		return
	}

	// 根据文件扩展名设置Content-Type
	ext := filepath.Ext(imageName)
	switch ext {
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".png":
		w.Header().Set("Content-Type", "image/png")
	case ".gif":
		w.Header().Set("Content-Type", "image/gif")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Write(imageData)
})

	fmt.Printf("wechatmp2markdown server listening on %s\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

var defHTML string = `
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>wechatmp2markdown</title>
</head>
<body>
	<h1 style="text-align: center; width: 100%;">wechatmp2markdown</h1>
	<ul style="margin: 0 auto; width: 89%;">
		<li>
			<strong>param 'url' is required.</strong> please put in a wechatmp URL and try again.
		</li>
		<li>
			<strong>param 'image' is optional</strong>, value include: 'url' / 'save' / 'base64'(default)
		</li>
		<li>
			<strong>example:</strong> http://localhost:8964/?url=https://mp.weixin.qq.com/s?__biz=aaaa==&mid=1111&idx=2&sn=bbbb&chksm=cccc&scene=123&image=save
		</li>
	</ul>
</body>
</html>
`

func parseParams(rawQuery string) map[string]string {
	result := make(map[string]string)
	reg := regexp.MustCompile(`(&?image=)([a-z]+)`)
	matcheImage := reg.FindStringSubmatch(rawQuery)
	var urlParamFull string = rawQuery
	if len(matcheImage) > 1 {
		// 有image参数
		imageParamFull := matcheImage[0]
		urlParamFull = strings.Replace(rawQuery, imageParamFull, "", 1)

		if len(matcheImage) > 2 {
			imageParamVal := matcheImage[2]
			result["image"] = imageParamVal
		}
	}
	regUrl := regexp.MustCompile(`(&?url=)(.+)`)
	matcheUrl := regUrl.FindStringSubmatch(urlParamFull)
	if len(matcheUrl) > 2 {
		urlParamVal := matcheUrl[2]
		result["url"] = urlParamVal
	}
	return result
}
