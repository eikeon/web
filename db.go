package web

import (
	"log"
	"strings"
	"time"

	"github.com/eikeon/dynamodb"
)

var db dynamodb.DynamoDB

func createTable(name string, i interface{}) {
	db = dynamodb.NewDynamoDB()
	if db != nil {
		t, err := db.Register(name, i)
		if err != nil {
			panic(err)
		}
		pt := dynamodb.ProvisionedThroughput{ReadCapacityUnits: 1, WriteCapacityUnits: 1}
		if _, err := db.CreateTable(t.TableName, t.AttributeDefinitions, t.KeySchema, pt, nil); err != nil {
			log.Println("CreateTable:", err)
		}
		for {
			if description, err := db.DescribeTable(name, nil); err != nil {
				log.Println("DescribeTable err:", err)
			} else {
				log.Println(description.Table.TableStatus)
				if description.Table.TableStatus == "ACTIVE" {
					break
				}
			}
			time.Sleep(time.Second)
		}
	} else {
		log.Println("WARNING: could not create database to persist stories.")
	}
}

func init() {
	createTable("resource", (*Resource)(nil))
}

func Get(url string) (*Resource, error) {
	if f, err := db.GetItem("resource", db.ToKey(&Resource{URL: url}), nil); err == nil {
		if f.Item != nil {
			return db.FromItem("resource", *f.Item).(*Resource), nil
		} else {
			if strings.HasSuffix(url, "/") == false {
				nurl := url + "/"
				if nr, err := Get(nurl); err == nil {
					if nr.Name != "404" {
						return &Resource{URL: url, Title: "See Other", Redirect: nurl}, nil
					}
				}
			}
			return &Resource{URL: url, Title: "Not Found", Name: "404"}, nil
		}
	} else {
		return nil, err
	}
}

func Put(r *Resource) {
	db.PutItem("resource", db.ToItem(r), nil)
}
