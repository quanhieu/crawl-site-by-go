package main

import (
	"crawl/business"
	"crawl/database"
	"crawl/models"
	"crawl/pkg"
	"crawl/pkg/crawl"
	articleStorage "crawl/storage"
	"crawl/util"
	"fmt"
	"strings"
)

//const PAGE_LASTER = "latest"
const DOMAIN_CRAWL string = "https://dev.to"

func main() {
	config, err := util.LoadConfig(".")
	if err != nil {
		fmt.Println("not load config", err)
		panic(err)
	}

	db, err := database.DBConn(config)
	if err != nil {
		panic(err)
	}
	storage := articleStorage.NewMySQLStorage(db)
	biz := business.NewArticleBusiness(storage)

	bizTag := business.NewTagBusiness(storage)

	articleTagBiz := business.NewArticleTagBusiness(storage)

	devToChan := make(chan []crawl.DataArticle)
	hashNode := make(chan []crawl.DataArticle)
	webFreeCodeCamp := make(chan []crawl.DataArticle)
	medium := make(chan []crawl.DataArticle)

	go crawl.CrawlWeb(devToChan)
	go crawl.CrawlWebFreeCodeCamp(webFreeCodeCamp)
	go crawl.CrawlWebMedium(medium)
	go crawl.CrawlWebHashNode(hashNode)

	insertData(config, <-webFreeCodeCamp, biz, bizTag, articleTagBiz)
	insertData(config, <-devToChan, biz, bizTag, articleTagBiz)
	insertData(config, <-medium, biz, bizTag, articleTagBiz)
	insertData(config, <-hashNode, biz, bizTag, articleTagBiz)
}

func insertData(config util.Config, dataResult []crawl.DataArticle, biz *business.ArticleBusiness, bizTag *business.TagBusiness, articleTagBiz *business.ArticleTagBusiness) {
	count := 0
	for _, data := range dataResult {
		if len(data.Tags) > 0 {
			for _, dataTag := range data.Tags {
				tag, err := bizTag.FindTag(map[string]interface{}{"slug": dataTag.Slug})
				if err != nil {
					tag := models.Tag{
						Title: dataTag.Title,
						Slug:  dataTag.Slug,
					}
					bizTag.CreateTag(tag)
				} else {
					bizTag.UpdateTag(map[string]interface{}{"slug": dataTag.Slug}, *tag)
				}
			}
		}

		check := strings.Contains(data.Slug, "go")
		if check && count < 5 {
			pkg.BotPushNewGoToDiscord(config, data.Title, data.Link, data.Image)
		}

		article, err := biz.FindArticle(map[string]interface{}{"slug": data.Slug})
		if err != nil {
			//fmt.Println("insert article: ", data.Title)
			article := models.Article{
				Title: data.Title,
				Slug:  data.Slug,
				Image: data.Image,
				Link:  data.Link,
			}
			biz.CreateArticle(&article)
			// insert article_tag
			if len(data.Tags) > 0 {
				for _, dataTag := range data.Tags {
					tag, err := bizTag.FindTag(map[string]interface{}{"slug": dataTag.Slug})
					if err == nil {
						// insert article with tag
						articleTag := models.ArticleTag{
							ArticleId: article.Id,
							TagId:     tag.Id,
						}
						articleTagBiz.CreateArticleTag(&articleTag)
					}
				}
			}
		} else {
			//fmt.Println("update article: ", article.Title)
			biz.UpdateArticle(map[string]interface{}{"slug": data.Slug}, *article)
		}

		count++
	}
}
