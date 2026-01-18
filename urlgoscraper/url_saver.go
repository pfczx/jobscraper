package urlsgocraper

import (
	"os"
	"strings"
)
func SaveUrls(filename string,data []string) error {
	content := strings.Join(data,"\n")
	return os.WriteFile("./"+filename,[]byte(content),0666)
}

func LoadUrls(filename string) ([]string,error){
	bytes,err := os.ReadFile(filename)
	if err !=nil{
		return nil,err
	}

	urls := strings.TrimSpace(string(bytes))
	if urls == ""{
		return []string{},nil
	}

	return strings.Split(urls,"\n"),nil
}
