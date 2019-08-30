package main

import (
	//"fmt"
	"encoding/base64"
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lestrrat/go-libxml2"
)

var proxyStr = "http://10.144.1.10:8080"
var proxyURL *url.URL
var rootURL = "http://m.php06.com"
var rootPicURL = "http://www.dc619.com"

func getLinkRepo(data string) string {
	strList := strings.Split(data, "\n")
	for _, s := range strList {
		if strings.Contains(s, "qTcms_S_m_murl_e") {
			varDateList := strings.Split(s, "\"")
			return varDateList[len(varDateList)-2]
		}
	}
	return ""
}
func decode(base64String string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(base64String)
}
func parseRepoListString(RepoListString string) []string {
	RepoList := strings.Split(RepoListString, "$qingtiandy$")
	return RepoList
}

func httpReqGet(urlStr, refer string) (*http.Response, error) {
	log.Println("URL=", urlStr, "  refer=", refer)
	url, err := url.Parse(urlStr)
	if err != nil {
		log.Println(err)
	}
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}
	client := &http.Client{
		Transport: transport,
	}
	request, err := http.NewRequest("GET", url.String(), nil)
	request.Close = true
	if err != nil {
		log.Println(err)
	}
	if refer != "" {
		request.Header.Set("Referer", refer)
	}
	return client.Do(request)
}

func httpGetChart(chartWg *sync.WaitGroup, urlStr string, chartIndex int, title string) error {
	defer chartWg.Done()
	response, err := httpReqGet(urlStr+"?p=1", "")
	if err != nil {
		return err
	}
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	repo, err := decode(getLinkRepo(string(data)))
	if err != nil {
		return err
	}
	parseRepoList := parseRepoListString(string(repo))
	log.Println("get parseRepoList: ", parseRepoList)
	var jieWg sync.WaitGroup
	for index, s := range parseRepoList {
		indexString := strconv.Itoa(index + 1)
		filePath := title + "/" + strconv.Itoa(chartIndex) + "/" + indexString + ".jpg"
		refer := strings.Replace(urlStr, "p=1", "p="+indexString, 1)
		jieWg.Add(1)
		go func(s string) {
			err := httpPicGetAndSave(&jieWg, s, refer, filePath)
			if err != nil {
				log.Println("download "+title+" chart "+strconv.Itoa(chartIndex)+" pic "+indexString+" from "+s+" failed", err)
			} else {
				log.Println("download " + title + " chart " + strconv.Itoa(chartIndex) + " pic " + indexString + " from " + s + " finish")
			}
		}(s)
	}
	jieWg.Wait()

	log.Println("download " + title + " chart " + strconv.Itoa(chartIndex) + " finish")
	return nil
}

func httpPicGetAndSave(jieWg *sync.WaitGroup, urlStr, refer, filePath string) error {
	defer jieWg.Done()
	urlStr = rootPicURL + urlStr
	response, err := httpReqGet(urlStr, refer)
	if err != nil || response == nil {
		return err
	}
	log.Println(response.StatusCode)
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return errors.New("http StatusCode " + strconv.Itoa(response.StatusCode))
	}
	pathSlice := strings.Split(filePath, "/")[:]
	folderPath := strings.Join(pathSlice[:len(pathSlice)-1], "/")
	err = os.MkdirAll(folderPath, os.ModePerm)
	if err != nil {
		return err
	}
	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, response.Body)
	if err != nil {
		return err
	}
	log.Println(urlStr + " save successfully")
	return nil
}

func httpGetChartList(urlStr string) ([]string, error) {
	response, err := httpReqGet(urlStr, "")
	if err != nil {
		return []string{""}, err
	}
	if doc, err := libxml2.ParseHTMLReader(response.Body); err != nil {
		return []string{""}, err
	} else {
		defer doc.Free()
		exprString := `//*[@id="mh-chapter-list-ol-0"]/li[*]/a`
		nodes, err := doc.Find(exprString)
		if err != nil {
			return []string{""}, err
		}
		var hrefList []string
		for _, node := range nodes.NodeList() {
			strList := strings.Split(node.String(), "\"")
			href := strList[1]
			hrefList = append(hrefList, href)
		}
		return hrefList, nil
	}
}

func f() {
	for {
		time.Sleep(100 * time.Millisecond)
		log.Println("current routine Num: ", runtime.NumGoroutine())
	}
}
func main() {
	//need exec ulimit -n 10240 ,otherwise too many open file error
	var err error
	titleUrl := flag.String("title", "/lianai/wodemiminvyou/", "title URL")
	proxy := flag.String("proxy", proxyStr, "proxy server URL")
	flag.Parse()
	go f()
	proxyURL, err = url.Parse(*proxy)
	if err != nil {
		log.Println(err)
	}
	titleFullUrl := rootURL + *titleUrl
	title := strings.Split(titleFullUrl, "/")[4]
	log.Println("start download " + title)
	ChartList, err := httpGetChartList(titleFullUrl)
	if err != nil {
		log.Println(err)
	}
	log.Println("get Chart List:", ChartList)
	if err != nil {
		log.Println(err)
	}
	var chartWg sync.WaitGroup
	for index, charURL := range ChartList {
		chartWg.Add(1)
		go httpGetChart(&chartWg, rootURL+charURL, index, title)
	}
	chartWg.Wait()
}
